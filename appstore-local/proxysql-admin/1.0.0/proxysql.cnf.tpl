datadir="/var/lib/proxysql"

admin_variables={
    # admin 用户名被 ProxySQL 强制只能本机登录，面板跨容器连接必须用第二个账号 uiadmin
    admin_credentials="admin:__ADMIN_PWD__;uiadmin:__ADMIN_PWD__"
    mysql_ifaces="0.0.0.0:6032"
}

mysql_variables={
    threads=4
    max_connections=2048
    interfaces="0.0.0.0:6033"
    monitor_username="monitor"
    monitor_password="__MONITOR_PWD__"
    monitor_connect_interval=60000
}

# 注意：mysql_servers / mysql_users 里有多个条目时，条目之间要用英文逗号分隔
mysql_servers=(
    { address="__BACKEND_HOST__", port=__BACKEND_PORT__, hostgroup=1, max_connections=200 }
)

mysql_users=(
    { username="webuser", password="ChangeMe_AppPwd", default_hostgroup=1, active=1 }
)
