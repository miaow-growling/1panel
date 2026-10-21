const express = require('express');
const crypto = require('crypto');
const fs = require('fs');
const mysql = require('mysql2/promise');
const path = require('path');

const {
  PSQL_ADMIN_HOST = '127.0.0.1',
  PSQL_ADMIN_PORT = '6032',
  PSQL_ADMIN_USER = 'admin',
  PSQL_ADMIN_PASSWORD = '',
  UI_PASSWORD = '',
  PORT = '3000',
  // 可选：配置后端 MySQL 管理账号后，面板可自动在后端创建账号并授权
  MYSQL_MGMT_HOST = '',
  MYSQL_MGMT_PORT = '3306',
  MYSQL_MGMT_USER = '',
  MYSQL_MGMT_PASSWORD = '',
} = process.env;

if (!PSQL_ADMIN_PASSWORD) {
  console.error('[启动失败] 未设置环境变量 PSQL_ADMIN_PASSWORD（ProxySQL 管理口密码）');
  process.exit(1);
}

let SECRET = crypto.randomBytes(32).toString('hex'); // 修改登录密码后轮换，使所有旧会话失效
const COOKIE = 'psql_admin_token';
const TOKEN_TTL_MS = 30 * 24 * 3600 * 1000;

// 面板数据目录与登录密码哈希：首次启动用 UI_PASSWORD 初始化，之后以文件为准（支持面板内修改）
const DATA_DIR = process.env.PANEL_DATA_DIR || path.join(__dirname, 'data');
const AUTH_FILE = path.join(DATA_DIR, 'auth.json');
const sha256hex = (s) => crypto.createHash('sha256').update(String(s)).digest('hex');
let authHash = null;
try { authHash = JSON.parse(fs.readFileSync(AUTH_FILE, 'utf8')).hash; } catch { /* 首次运行 */ }
if (!authHash) {
  if (!UI_PASSWORD) {
    console.error('[启动失败] 首次启动需设置环境变量 UI_PASSWORD（面板初始登录密码，可用 install.sh 自动生成）');
    process.exit(1);
  }
  authHash = sha256hex(UI_PASSWORD);
  fs.mkdirSync(DATA_DIR, { recursive: true });
  fs.writeFileSync(AUTH_FILE, JSON.stringify({ hash: authHash }, null, 2));
  console.log('[初始化] 面板登录密码已初始化并写入数据文件');
}

// ProxySQL 管理口(6032)讲的是 MySQL 协议，用 mysql2 直连即可
const pool = mysql.createPool({
  host: PSQL_ADMIN_HOST,
  port: Number(PSQL_ADMIN_PORT),
  user: PSQL_ADMIN_USER,
  password: PSQL_ADMIN_PASSWORD,
  database: 'main',
  connectionLimit: 4,
  multipleStatements: false,
});

async function q(sql, values = []) {
  const [rows] = await pool.query(sql, values);
  return rows;
}

async function applyGroup(group) {
  await q(`LOAD ${group} TO RUNTIME`);
  await q(`SAVE ${group} TO DISK`);
}

// ---------- 登录态（HMAC 签名 Cookie） ----------
const sign = (ts) => crypto.createHmac('sha256', SECRET).update(String(ts)).digest('hex');
const makeToken = () => {
  const ts = Date.now();
  return `${ts}.${sign(ts)}`;
};
function verifyToken(t) {
  if (!t) return false;
  const [ts, sig] = String(t).split('.');
  if (!ts || !sig || sig.length !== 64) return false;
  try {
    if (!crypto.timingSafeEqual(Buffer.from(sig), Buffer.from(sign(ts)))) return false;
  } catch {
    return false;
  }
  return Date.now() - Number(ts) < TOKEN_TTL_MS;
}
function getCookie(req, name) {
  for (const part of (req.headers.cookie || '').split(';')) {
    const i = part.indexOf('=');
    if (i > 0 && part.slice(0, i).trim() === name) return decodeURIComponent(part.slice(i + 1).trim());
  }
  return null;
}

const app = express();
app.use(express.json());

app.post('/api/login', (req, res) => {
  let ok = false;
  try {
    ok = crypto.timingSafeEqual(Buffer.from(sha256hex(req.body?.password || '')), Buffer.from(authHash));
  } catch { ok = false; }
  if (!ok) {
    return setTimeout(() => res.status(401).json({ error: '密码错误' }), 600);
  }
  res.setHeader('Set-Cookie', `${COOKIE}=${makeToken()}; Path=/; HttpOnly; SameSite=Lax; Max-Age=${TOKEN_TTL_MS / 1000}`);
  res.json({ ok: true });
});

app.post('/api/logout', (req, res) => {
  res.setHeader('Set-Cookie', `${COOKIE}=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0`);
  res.json({ ok: true });
});

app.use('/api', (req, res, next) => {
  if (verifyToken(getCookie(req, COOKIE))) return next();
  res.status(401).json({ error: '未登录或登录已过期' });
});

const wrap = (fn) => (req, res) =>
  fn(req, res).catch((e) => {
    console.error('[api]', e.message);
    res.status(500).json({ error: e.message });
  });

// 修改面板登录密码（成功后轮换会话密钥，所有旧登录失效，需用新密码重新登录）
app.post('/api/auth/change', wrap(async (req, res) => {
  const b = req.body || {};
  let oldOk = false;
  try {
    oldOk = crypto.timingSafeEqual(Buffer.from(sha256hex(b.oldPassword || '')), Buffer.from(authHash));
  } catch { oldOk = false; }
  if (!oldOk) return res.status(400).json({ error: '当前密码错误' });
  const np = String(b.newPassword || '');
  if (np.length < 8 || np.length > 64) return res.status(400).json({ error: '新密码长度需为 8-64 位' });
  const newHash = sha256hex(np);
  fs.mkdirSync(DATA_DIR, { recursive: true });
  fs.writeFileSync(AUTH_FILE, JSON.stringify({ hash: newHash }, null, 2));
  authHash = newHash;
  SECRET = crypto.randomBytes(32).toString('hex');
  res.json({ ok: true });
}));

// ---------- 后端 MySQL 实例管理账号（用于自动建号授权） ----------
// 凭据来源：面板本机数据文件（每个实例单独配置）+ 环境变量默认实例（MYSQL_MGMT_*）
const MGMT_FILE = path.join(DATA_DIR, 'mgmt-instances.json');
let mgmtInstances = {};   // key: "host:port" -> { host, port, user, password }
try { mgmtInstances = JSON.parse(fs.readFileSync(MGMT_FILE, 'utf8')); } catch { /* 首次运行无文件 */ }
function saveInstances() {
  fs.mkdirSync(DATA_DIR, { recursive: true });
  fs.writeFileSync(MGMT_FILE, JSON.stringify(mgmtInstances, null, 2));
}
const instKey = (h, p) => `${String(h)}:${Number(p)}`;
function resolveMgmt(host, port) {
  const k = instKey(host, port);
  if (mgmtInstances[k]) return mgmtInstances[k];
  if (MYSQL_MGMT_HOST && k === instKey(MYSQL_MGMT_HOST, MYSQL_MGMT_PORT)) {
    return { host: MYSQL_MGMT_HOST, port: Number(MYSQL_MGMT_PORT), user: MYSQL_MGMT_USER, password: MYSQL_MGMT_PASSWORD };
  }
  return null;
}
const mgmtPools = new Map();
function mgmt(cred) {
  const k = instKey(cred.host, cred.port);
  if (!mgmtPools.has(k)) {
    // key 用后端数据库页的地址；实际连接地址可在配置管理账号时单独指定（面板与 ProxySQL 网络视角可能不同）
    mgmtPools.set(k, mysql.createPool({
      host: cred.connectHost || cred.host, port: Number(cred.connectPort || cred.port),
      user: cred.user, password: cred.password,
      connectionLimit: 2, multipleStatements: false,
    }));
  }
  return mgmtPools.get(k);
}
const idq = (s) => '`' + String(s).replace(/`/g, '``') + '`';
const PRIV_SQL = { read: 'SELECT', write: 'SELECT, INSERT, UPDATE, DELETE', all: 'ALL PRIVILEGES' };
const SYS_DBS = new Set(['information_schema', 'mysql', 'performance_schema', 'sys']);

app.get('/api/mgmt/status', wrap(async (req, res) => {
  res.json({ configured: !!(Object.keys(mgmtInstances).length || (MYSQL_MGMT_HOST && MYSQL_MGMT_USER)) });
}));

app.get('/api/mgmt/instances', wrap(async (req, res) => {
  const list = Object.values(mgmtInstances).map((i) => ({ host: i.host, port: i.port, source: 'file' }));
  if (MYSQL_MGMT_HOST && MYSQL_MGMT_USER && !mgmtInstances[instKey(MYSQL_MGMT_HOST, MYSQL_MGMT_PORT)]) {
    list.unshift({ host: MYSQL_MGMT_HOST, port: Number(MYSQL_MGMT_PORT), source: 'env' });
  }
  res.json({ instances: list });
}));

app.post('/api/mgmt/instances', wrap(async (req, res) => {
  const b = req.body || {};
  if (!b.host || !b.user) return res.status(400).json({ error: '实例地址和管理用户必填' });
  const cred = {
    host: String(b.host), port: Number(b.port) || 3306,
    connectHost: String(b.connectHost || b.host), connectPort: Number(b.connectPort || b.port) || 3306,
    user: String(b.user), password: String(b.password || ''),
  };
  // 编辑时密码留空 = 沿用已保存的密码
  if (!cred.password && mgmtInstances[instKey(cred.host, cred.port)]) {
    cred.password = mgmtInstances[instKey(cred.host, cred.port)].password;
  }
  const test = mysql.createPool({ host: cred.connectHost, port: cred.connectPort, user: cred.user, password: cred.password, connectionLimit: 1, multipleStatements: false });
  try {
    await test.query('SELECT 1');
  } catch (e) {
    await test.end().catch(() => {});
    return res.status(400).json({ error: '连接失败：' + e.message });
  }
  await test.end().catch(() => {});
  mgmtInstances[instKey(cred.host, cred.port)] = cred;
  saveInstances();
  res.json({ ok: true });
}));

app.delete('/api/mgmt/instances', wrap(async (req, res) => {
  const k = instKey(req.query.host, req.query.port);
  if (!mgmtInstances[k]) return res.status(404).json({ error: '实例不存在' });
  delete mgmtInstances[k];
  saveInstances();
  res.json({ ok: true });
}));

app.get('/api/mgmt/databases', wrap(async (req, res) => {
  const cred = resolveMgmt(req.query.host, req.query.port);
  if (!cred) return res.status(400).json({ error: '该实例未配置管理账号，请先在后端数据库页设置' });
  const [rows] = await mgmt(cred).query('SHOW DATABASES');
  res.json(rows.map((r) => String(r.Database)).filter((d) => !SYS_DBS.has(d.toLowerCase())));
}));

app.post('/api/mgmt/grant', wrap(async (req, res) => {
  const b = req.body || {};
  const cred = resolveMgmt(b.host, b.port);
  if (!cred) return res.status(400).json({ error: '该实例未配置管理账号，请先在后端数据库页设置' });
  if (!b.username) return res.status(400).json({ error: '缺少用户名' });
  if (!PRIV_SQL[b.privilege]) return res.status(400).json({ error: '权限级别无效' });
  const dbs = Array.isArray(b.databases) && b.databases.length ? b.databases.map(String) : (b.database ? [String(b.database)] : []);
  if (!dbs.length) return res.status(400).json({ error: '至少选择一个要授权的数据库' });
  const conn = mgmt(cred);
  const host = '%';
  const [urows] = await conn.query('SELECT COUNT(*) AS n FROM mysql.user WHERE user=? AND host=?', [String(b.username), host]);
  const exists = urows[0].n > 0;
  if (exists) {
    if (b.password) {
      await conn.query('ALTER USER ??@?? IDENTIFIED WITH mysql_native_password BY ?', [String(b.username), host, String(b.password)]);
    }
  } else {
    if (!b.password) return res.status(400).json({ error: '后端新建账号需要设置密码' });
    await conn.query('CREATE USER ??@?? IDENTIFIED WITH mysql_native_password BY ?', [String(b.username), host, String(b.password)]);
  }
  const [drows] = await conn.query('SHOW DATABASES');
  const names = drows.map((r) => String(r.Database));
  const scopes = dbs.includes('*') ? ['*.*'] : dbs.map((d) => {
    if (!names.includes(d)) throw new Error(`数据库 ${d} 在该实例不存在`);
    return `${idq(d)}.*`;
  });
  try {
    // 先清掉旧授权再重新授，保证权限/库选择变更后不留残留
    await conn.query('REVOKE ALL PRIVILEGES, GRANT OPTION ON *.* FROM ??@??', [String(b.username), host]);
  } catch { /* 尚无授权时会报错，忽略 */ }
  for (const scope of scopes) {
    await conn.query(`GRANT ${PRIV_SQL[b.privilege]} ON ${scope} TO ??@??`, [String(b.username), host]);
  }
  res.json({ ok: true, created: !exists, scopes });
}));

// ---------- 概览 ----------
app.get('/api/overview', wrap(async (req, res) => {
  const [version] = await q("SELECT variable_value AS v FROM global_variables WHERE variable_name='admin-version'");
  const globals = await q(
    "SELECT Variable_Name AS name, Variable_Value AS value FROM stats_mysql_global WHERE Variable_Name IN ('Questions','Slow_queries','Client_Connections_connected','Client_Connections_aborted')"
  );
  const pools = await q(
    'SELECT hostgroup, srv_host, srv_port, status, ConnUsed, ConnFree, Queries FROM stats_mysql_connection_pool ORDER BY CAST(hostgroup AS INTEGER), srv_host, srv_port'
  );
  let userStats = [];
  try {
    userStats = await q('SELECT username, frontend_connections, backend_connections FROM stats_mysql_users_statuses');
  } catch {
    /* 低版本无此表，忽略 */
  }
  res.json({ version: version ? version.v : '未知', globals, pools, userStats });
}));

// ---------- 后端数据库 ----------
app.get('/api/servers', wrap(async (req, res) => {
  const rows = await q(
    'SELECT hostgroup_id, hostname, port, status, weight, max_connections, max_replication_lag, comment FROM mysql_servers ORDER BY hostgroup_id, hostname, port'
  );
  res.json(rows.map((r) => {
    const inst = mgmtInstances[instKey(r.hostname, r.port)];
    return { ...r, mgmt: !!inst, mgmtUser: inst ? inst.user : '' };
  }));
}));

app.post('/api/servers', wrap(async (req, res) => {
  const b = req.body || {};
  if (!b.hostname || !b.port) return res.status(400).json({ error: '地址和端口必填' });
  await q(
    'INSERT INTO mysql_servers (hostgroup_id, hostname, port, status, weight, max_connections, max_replication_lag, comment) VALUES (?,?,?,?,?,?,?,?)',
    [Number(b.hostgroup_id) || 1, String(b.hostname), Number(b.port), b.status || 'ONLINE',
     Number(b.weight) || 1, Number(b.max_connections) || 200, Number(b.max_replication_lag) || 0, b.comment || '']
  );
  await applyGroup('MYSQL SERVERS');
  res.json({ ok: true });
}));

app.put('/api/servers', wrap(async (req, res) => {
  const b = req.body || {};
  if (!b.hostname || !b.port) return res.status(400).json({ error: '缺少主键字段' });
  const sets = [];
  const vals = [];
  for (const k of ['status', 'weight', 'max_connections', 'max_replication_lag', 'comment']) {
    if (b[k] === undefined) continue;
    sets.push(`${k}=?`);
    vals.push(k === 'status' || k === 'comment' ? String(b[k] ?? '') : Number(b[k]) || 0);
  }
  if (!sets.length) return res.status(400).json({ error: '没有要更新的字段' });
  vals.push(Number(b.hostgroup_id), String(b.hostname), Number(b.port));
  await q(`UPDATE mysql_servers SET ${sets.join(', ')} WHERE hostgroup_id=? AND hostname=? AND port=?`, vals);
  await applyGroup('MYSQL SERVERS');
  res.json({ ok: true });
}));

app.delete('/api/servers', wrap(async (req, res) => {
  await q('DELETE FROM mysql_servers WHERE hostgroup_id=? AND hostname=? AND port=?',
    [Number(req.query.hostgroup_id), String(req.query.hostname), Number(req.query.port)]);
  await applyGroup('MYSQL SERVERS');
  res.json({ ok: true });
}));

// ---------- 账号 ----------
app.get('/api/users', wrap(async (req, res) => {
  res.json(await q(
    'SELECT username, active, backend, frontend, default_hostgroup, default_schema, transaction_persistent, fast_forward, max_connections, comment FROM mysql_users ORDER BY username'
  ));
}));

// 单个账号的连接信息（含密码明文；若是哈希则无法回显）
app.get('/api/users/conninfo', wrap(async (req, res) => {
  const name = String(req.query.username || '');
  if (!name) return res.status(400).json({ error: '缺少用户名' });
  const rows = await q('SELECT username, password, default_schema, default_hostgroup FROM mysql_users WHERE username=?', [name]);
  if (!rows.length) return res.status(404).json({ error: '账号不存在' });
  const pw = String(rows[0].password || '');
  const hashed = pw.startsWith('*') || pw.startsWith('$A00$') || /^[0-9A-F]{80}$/.test(pw);
  res.json({
    username: rows[0].username,
    default_schema: rows[0].default_schema || '',
    default_hostgroup: rows[0].default_hostgroup,
    password: hashed ? '' : pw,
    passwordHashed: hashed,
    publicPort: Number(process.env.PSQL_PUBLIC_PORT || 6033),
  });
}));

app.post('/api/users', wrap(async (req, res) => {
  const b = req.body || {};
  if (!b.username) return res.status(400).json({ error: '用户名必填' });
  await q(
    'INSERT INTO mysql_users (username, password, active, default_hostgroup, default_schema, transaction_persistent, fast_forward, max_connections, comment) VALUES (?,?,?,?,?,?,?,?,?)',
    [String(b.username), String(b.password || ''), b.active === false ? 0 : 1,
     Number(b.default_hostgroup) || 1, b.default_schema || '', b.transaction_persistent ? 1 : 0,
     0, Number(b.max_connections) || 10000, b.comment || '']
  );
  await applyGroup('MYSQL USERS');
  res.json({ ok: true });
}));

app.put('/api/users', wrap(async (req, res) => {
  const b = req.body || {};
  if (!b.username) return res.status(400).json({ error: '缺少用户名' });
  // 只更新请求里出现的字段，避免部分编辑时把未传字段清空
  const sets = [];
  const vals = [];
  for (const k of ['active', 'default_hostgroup', 'transaction_persistent', 'max_connections']) {
    if (b[k] === undefined) continue;
    sets.push(`${k}=?`);
    vals.push(k === 'active' || k === 'transaction_persistent' ? (b[k] ? 1 : 0) : Number(b[k]) || 0);
  }
  for (const k of ['default_schema', 'comment']) {
    if (b[k] === undefined) continue;
    sets.push(`${k}=?`);
    vals.push(String(b[k] ?? ''));
  }
  if (b.password) {
    sets.push('password=?');
    vals.push(String(b.password));
  }
  if (!sets.length) return res.status(400).json({ error: '没有要更新的字段' });
  vals.push(String(b.username));
  await q(`UPDATE mysql_users SET ${sets.join(', ')} WHERE username=?`, vals);
  await applyGroup('MYSQL USERS');
  res.json({ ok: true });
}));

app.delete('/api/users', wrap(async (req, res) => {
  await q('DELETE FROM mysql_users WHERE username=?', [String(req.query.username)]);
  await applyGroup('MYSQL USERS');
  res.json({ ok: true });
}));

// ---------- 一键全部重载 ----------
app.post('/api/apply', wrap(async (req, res) => {
  for (const g of ['MYSQL SERVERS', 'MYSQL USERS', 'MYSQL QUERY RULES', 'MYSQL VARIABLES']) {
    await applyGroup(g);
  }
  res.json({ ok: true });
}));

// ---------- 静态页面 ----------
app.use('/api', (req, res) => res.status(404).json({ error: '接口不存在' }));
app.use(express.static(path.join(__dirname, 'public')));
app.get('*', (req, res) => res.sendFile(path.join(__dirname, 'public', 'index.html')));

app.listen(Number(PORT), '0.0.0.0', () => {
  console.log(`ProxySQL 控制台已启动: http://0.0.0.0:${PORT} -> ProxySQL 管理口 ${PSQL_ADMIN_HOST}:${PSQL_ADMIN_PORT}`);
});
