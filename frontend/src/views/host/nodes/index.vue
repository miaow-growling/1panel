<template>
    <LayoutContent :title="'集群节点管理'">
        <template #leftToolBar>
            <el-button type="primary" @click="openCreate">{{ '添加节点' }}</el-button>
            <el-button @click="search()">{{ '刷新' }}</el-button>
        </template>
        <template #rightToolBar>
            <TableSearch @search="search()" v-model:searchName="searchName" />
        </template>
        <template #main>
            <ComplexTable :data="nodes" @search="search()" v-loading="loading">
                <el-table-column :label="'节点名称'" min-width="140" fix>
                    <template #default="{ row }">
                        <span>{{ row.isLocal ? '主控节点 (本机)' : row.name }}</span>
                    </template>
                </el-table-column>
                <el-table-column :label="'通信地址'" min-width="180" prop="address" />
                <el-table-column :label="'状态'" min-width="80">
                    <template #default="{ row }">
                        <el-tag v-if="row.status === 'online'" type="success" size="small">在线</el-tag>
                        <el-tag v-else-if="row.status === 'installing'" type="warning" size="small">安装中</el-tag>
                        <el-tag v-else type="danger" size="small">离线</el-tag>
                    </template>
                </el-table-column>
                <el-table-column :label="'版本'" min-width="90">
                    <template #default="{ row }">
                        <span>{{ row.version || '-' }}</span>
                    </template>
                </el-table-column>
                <el-table-column :label="'备注'" min-width="120" prop="description" />
                <el-table-column :label="'操作'" min-width="200" fixed="right">
                    <template #default="{ row }">
                        <el-button v-if="!row.isLocal" text type="primary" size="small" @click="onTest(row)">
                            {{ '连通测试' }}
                        </el-button>
                        <el-button v-if="!row.isLocal" text type="primary" size="small" @click="openEdit(row)">
                            {{ '编辑' }}
                        </el-button>
                        <el-button v-if="!row.isLocal" text type="danger" size="small" @click="onDelete(row)">
                            {{ '删除' }}
                        </el-button>
                    </template>
                </el-table-column>
            </ComplexTable>
        </template>
    </LayoutContent>

    <DialogPro v-model="dialogVisible" :title="isEdit ? '编辑节点' : '添加节点'" width="640px">
        <el-tabs v-if="!isEdit" v-model="activeTab">
            <el-tab-pane label="一键安装（推荐）" name="script">
                <el-form label-width="100px">
                    <el-form-item :label="'节点 IP'" required>
                        <el-input v-model="scriptForm.nodeIP" placeholder="主控可达的节点 IP，如 10.0.0.2" />
                    </el-form-item>
                    <el-form-item :label="'节点名称'">
                        <el-input v-model="scriptForm.name" placeholder="留空自动生成，如 node-10-0-0-2" />
                    </el-form-item>
                    <el-form-item :label="'Agent 端口'">
                        <el-input-number v-model="scriptForm.port" :min="1024" :max="65535" />
                    </el-form-item>
                    <el-form-item :label="'通信密钥'">
                        <el-input v-model="scriptForm.secretToken" show-password>
                            <template #append>
                                <el-button @click="genToken">{{ '随机生成' }}</el-button>
                            </template>
                        </el-input>
                    </el-form-item>
                    <el-form-item>
                        <el-button type="primary" :loading="scriptLoading" @click="genScript">
                            {{ '生成安装命令' }}
                        </el-button>
                    </el-form-item>
                    <el-form-item v-if="installScript">
                        <el-input
                            type="textarea"
                            :rows="8"
                            readonly
                            :model-value="installScript"
                        />
                    </el-form-item>
                    <el-form-item v-if="installScript">
                        <el-button type="success" @click="copyScript">{{ '复制命令' }}</el-button>
                        <el-alert
                            style="margin-left: 12px; flex: 1"
                            type="success"
                            :closable="false"
                            title="节点已自动登记（状态：安装中）。在目标服务器以 root 执行该命令后，Agent 上线会自动转为在线，无需手动登记。"
                        />
                    </el-form-item>
                </el-form>
            </el-tab-pane>
            <el-tab-pane label="手动添加" name="manual">
                <el-form ref="formRef" :model="form" label-width="100px">
                    <el-form-item :label="'节点名称'" required>
                        <el-input v-model="form.name" placeholder="agent-node-1" />
                    </el-form-item>
                    <el-form-item :label="'通信地址'" required>
                        <el-input v-model="form.address" placeholder="https://10.0.0.2:9998" />
                    </el-form-item>
                    <el-form-item :label="'通信密钥'" required>
                        <el-input v-model="form.secretToken" show-password placeholder="对端 PANEL_AGENT_TOKEN 的值" />
                    </el-form-item>
                    <el-form-item :label="'备注'">
                        <el-input v-model="form.description" />
                    </el-form-item>
                </el-form>
            </el-tab-pane>
        </el-tabs>
        <el-form v-else ref="formRef" :model="form" label-width="100px">
            <el-form-item :label="'节点名称'" required>
                <el-input v-model="form.name" placeholder="agent-node-1" />
            </el-form-item>
            <el-form-item :label="'通信地址'" required>
                <el-input v-model="form.address" placeholder="https://10.0.0.2:9998" />
            </el-form-item>
            <el-form-item :label="'通信密钥'" required>
                <el-input v-model="form.secretToken" show-password placeholder="对端 PANEL_AGENT_TOKEN 的值" />
            </el-form-item>
            <el-form-item :label="'备注'">
                <el-input v-model="form.description" />
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="dialogVisible = false">{{ '取消' }}</el-button>
            <el-button v-if="activeTab === 'manual' || isEdit" type="primary" :loading="submitting" @click="onSubmit">
                {{ '确定' }}
            </el-button>
        </template>
    </DialogPro>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import {
    listHostNodes,
    createHostNode,
    updateHostNode,
    deleteHostNode,
    testHostNodeByID,
    getInstallScript,
} from '@/api/modules/host-node';
import { MsgSuccess } from '@/utils/message';
import { ElMessageBox } from 'element-plus';
import { useGlobalStore } from '@/composables/useGlobalStore';
import router from '@/routers';

const loading = ref(false);
const submitting = ref(false);
const dialogVisible = ref(false);
const isEdit = ref(false);
const activeTab = ref('script');
const searchName = ref('');
const nodes = ref<any[]>([]);
const form = ref({ id: 0, name: '', address: '', secretToken: '', description: '' });
const scriptForm = ref({ nodeIP: '', name: '', port: 9998, secretToken: '' });
const installScript = ref('');
const scriptLoading = ref(false);

const search = async () => {
    loading.value = true;
    try {
        const res = await listHostNodes();
        nodes.value = (res.data || []).filter(
            (n: any) => !searchName.value || n.name.includes(searchName.value) || (n.address || '').includes(searchName.value),
        );
    } finally {
        loading.value = false;
    }
};

const genToken = () => {
    const arr = new Uint8Array(16);
    crypto.getRandomValues(arr);
    scriptForm.value.secretToken = Array.from(arr)
        .map((b) => b.toString(16).padStart(2, '0'))
        .join('');
};

const genScript = async () => {
    if (!scriptForm.value.nodeIP) {
        MsgError('请填写主控可达的节点 IP');
        return;
    }
    if (!scriptForm.value.secretToken) {
        genToken();
    }
    scriptLoading.value = true;
    try {
        const res = await getInstallScript({
            nodeIP: scriptForm.value.nodeIP,
            name: scriptForm.value.name,
            port: scriptForm.value.port,
            secretToken: scriptForm.value.secretToken,
        });
        installScript.value = res.data || '';
        if (scriptForm.value.nodeIP) {
            installScript.value = installScript.value.replace(
                '地址: https://<本机IP>:$AGENT_PORT',
                `地址: https://${scriptForm.value.nodeIP}:${scriptForm.value.port}`,
            );
        }
        search();
    } finally {
        scriptLoading.value = false;
    }
};

const copyScript = async () => {
    try {
        await navigator.clipboard.writeText(installScript.value);
        MsgSuccess('已复制到剪贴板');
    } catch {
        MsgSuccess('请手动选择文本复制');
    }
};

const openCreate = () => {
    isEdit.value = false;
    activeTab.value = 'script';
    form.value = { id: 0, name: '', address: '', secretToken: '', description: '' };
    if (!scriptForm.value.secretToken) {
        genToken();
    }
    installScript.value = '';
    dialogVisible.value = true;
};

const openEdit = (row: any) => {
    isEdit.value = true;
    form.value = { id: row.id, name: row.name, address: row.address, secretToken: '', description: row.description };
    dialogVisible.value = true;
};

const onSubmit = async () => {
    submitting.value = true;
    try {
        if (isEdit.value) {
            await updateHostNode(form.value);
        } else {
            await createHostNode(form.value);
        }
        MsgSuccess('操作成功');
        dialogVisible.value = false;
        search();
    } finally {
        submitting.value = false;
    }
};

const onTest = async (row: any) => {
    await testHostNodeByID({ id: row.id });
    MsgSuccess('连通测试通过');
};

const onDelete = async (row: any) => {
    await ElMessageBox.confirm(`确定删除节点 ${row.name} ？`, '提示', { type: 'warning' });
    await deleteHostNode(row.id);
    MsgSuccess('删除成功');
    search();
};

onMounted(() => {
    const { currentNode } = useGlobalStore();
    if (currentNode !== 'local') {
        MsgError('集群节点管理仅在主控节点可用');
        router.push('/');
        return;
    }
    search();
});
</script>
