<template>
    <div>
        <el-card class="mb-4">
            <template #header>
                <div class="flex justify-between items-center">
                    <span class="font-bold text-base">集群节点管理</span>
                    <el-button type="primary" @click="openCreateDialog">
                        <el-icon class="mr-1"><Plus /></el-icon>
                        添加被控节点
                    </el-button>
                </div>
            </template>
            <el-alert type="info" :closable="false" class="mb-4" show-icon>
                <template #title>
                    <strong>轻量化多节点架构说明：</strong>
                    主节点负责前端展示与统一调度，子节点仅需部署轻量
                    Agent（~35MB）。切换目标节点后，主界面能看、能管的所有功能（容器、文件、SSL、站点、数据库）无缝映射至对应子节点机器！
                </template>
            </el-alert>

            <div class="flex justify-between items-center mb-4">
                <div class="flex items-center gap-2">
                    <el-input
                        v-model="searchInfo"
                        placeholder="搜索节点名称 / 地址..."
                        clearable
                        style="width: 260px"
                        @clear="loadData"
                        @keyup.enter="loadData"
                    >
                        <template #prefix>
                            <el-icon><Search /></el-icon>
                        </template>
                    </el-input>
                    <el-button @click="loadData">查询</el-button>
                </div>
                <el-button @click="loadData">
                    <el-icon class="mr-1"><Refresh /></el-icon>
                    刷新状态
                </el-button>
            </div>

            <el-table :data="nodeList" v-loading="loading" border style="width: 100%">
                <el-table-column prop="name" label="节点名称" min-width="150">
                    <template #default="{ row }">
                        <div class="flex items-center gap-2">
                            <span class="font-medium">{{ row.name }}</span>
                            <el-tag v-if="row.id === 'local'" size="small" type="success">Master 主控</el-tag>
                            <el-tag v-else size="small" type="info">Worker 从机</el-tag>
                            <el-tag v-if="row.id === nodeStore.currentNode" size="small" effect="dark" type="warning">
                                当前选中
                            </el-tag>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column prop="address" label="通信地址" min-width="180" />
                <el-table-column prop="status" label="通信状态" width="120" align="center">
                    <template #default="{ row }">
                        <el-tag :type="row.status === 'online' ? 'success' : 'danger'">
                            {{ row.status === 'online' ? '在线' : '离线' }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="description" label="备注" min-width="150" show-overflow-tooltip />
                <el-table-column prop="createdAt" label="接入时间" width="180" />
                <el-table-column label="操作" width="260" fixed="right">
                    <template #default="{ row }">
                        <el-button
                            v-if="row.id !== nodeStore.currentNode"
                            type="primary"
                            link
                            size="small"
                            @click="switchCurrentNode(row.id)"
                        >
                            切换至此节点
                        </el-button>
                        <el-button
                            v-if="row.id !== 'local'"
                            type="info"
                            link
                            size="small"
                            :loading="testingId === row.id"
                            @click="testExistingNode(row)"
                        >
                            测速
                        </el-button>
                        <el-button
                            v-if="row.id !== 'local'"
                            type="warning"
                            link
                            size="small"
                            @click="openEditDialog(row)"
                        >
                            编辑
                        </el-button>
                        <el-popconfirm
                            v-if="row.id !== 'local'"
                            title="确定要移除该节点吗？移除后可通过重新添加恢复。"
                            @confirm="handleDelete(row.id)"
                        >
                            <template #reference>
                                <el-button type="danger" link size="small">删除</el-button>
                            </template>
                        </el-popconfirm>
                    </template>
                </el-table-column>
            </el-table>
        </el-card>

        <!-- 添加 / 编辑节点弹窗 -->
        <el-dialog
            v-model="dialogVisible"
            :title="isEdit ? '编辑被控节点' : '添加被控从机节点 (Worker Agent)'"
            width="560px"
            destroy-on-close
        >
            <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
                <el-form-item label="节点名称" prop="name">
                    <el-input v-model="form.name" placeholder="例如：香港边缘节点-01" />
                </el-form-item>
                <el-form-item label="通信地址" prop="address">
                    <el-input v-model="form.address" placeholder="例如：http://192.168.1.100:9999 或域名" />
                    <div class="text-xs text-gray-400 mt-1">需保证 Master 主节点网络能够直连该地址与端口</div>
                </el-form-item>
                <el-form-item label="通信密钥" prop="secretToken">
                    <el-input
                        v-model="form.secretToken"
                        placeholder="子节点 Agent 启动时配置的密钥 (Bearer Token)"
                        show-password
                    />
                </el-form-item>
                <el-form-item label="备注说明" prop="description">
                    <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选备注信息" />
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="flex justify-between items-center w-full">
                    <el-button :loading="testingConnection" @click="testNewConnection">测试连通性</el-button>
                    <div>
                        <el-button @click="dialogVisible = false">取消</el-button>
                        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
                    </div>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { listNodes, createNode, updateNode, deleteNode, testNode, Node } from '@/api/modules/node';
import { useNodeStore } from '@/store';
import { MsgSuccess, MsgError } from '@/utils/message';
import { Plus, Search, Refresh } from '@element-plus/icons-vue';
import type { FormInstance, FormRules } from 'element-plus';

const nodeStore = useNodeStore();
const nodeList = ref<Node.NodeInfo[]>([]);
const loading = ref(false);
const searchInfo = ref('');
const testingId = ref('');

const dialogVisible = ref(false);
const isEdit = ref(false);
const submitting = ref(false);
const testingConnection = ref(false);
const formRef = ref<FormInstance>();

const form = reactive({
    id: '',
    name: '',
    address: '',
    secretToken: '',
    description: '',
});

const rules = reactive<FormRules>({
    name: [{ required: true, message: '请输入节点名称', trigger: 'blur' }],
    address: [{ required: true, message: '请输入通信地址 (如 http://1.2.3.4:9999)', trigger: 'blur' }],
    secretToken: [{ required: true, message: '请输入通信密钥', trigger: 'blur' }],
});

const loadData = async () => {
    loading.value = true;
    try {
        const res = await listNodes();
        if (res && res.data) {
            let list = res.data;
            if (searchInfo.value.trim()) {
                const keyword = searchInfo.value.trim().toLowerCase();
                list = list.filter(
                    (n) =>
                        n.name.toLowerCase().includes(keyword) ||
                        n.address.toLowerCase().includes(keyword) ||
                        (n.description && n.description.toLowerCase().includes(keyword)),
                );
            }
            nodeList.value = list;
            nodeStore.nodeList = res.data;
        }
    } catch (err: any) {
        MsgError(err.message || '加载节点列表失败');
    } finally {
        loading.value = false;
    }
};

onMounted(() => {
    loadData();
});

const switchCurrentNode = (nodeId: string) => {
    nodeStore.switchNode(nodeId);
};

const testExistingNode = async (row: Node.NodeInfo) => {
    testingId.value = row.id;
    try {
        await testNode({
            address: row.address,
            secretToken: row.secretToken || '',
        });
        MsgSuccess(`节点 [${row.name}] 通信连通成功！`);
    } catch (err: any) {
        MsgError(`节点通信失败: ${err.message || '网络无法触达'}`);
    } finally {
        testingId.value = '';
    }
};

const openCreateDialog = () => {
    isEdit.value = false;
    form.id = '';
    form.name = '';
    form.address = '';
    form.secretToken = '';
    form.description = '';
    dialogVisible.value = true;
};

const openEditDialog = (row: Node.NodeInfo) => {
    isEdit.value = true;
    form.id = row.id;
    form.name = row.name;
    form.address = row.address;
    form.secretToken = '';
    form.description = row.description || '';
    dialogVisible.value = true;
};

const testNewConnection = async () => {
    if (!form.address) {
        MsgError('请先填写通信地址');
        return;
    }
    testingConnection.value = true;
    try {
        await testNode({
            address: form.address,
            secretToken: form.secretToken,
        });
        MsgSuccess('目标节点通信正常，握手成功！');
    } catch (err: any) {
        MsgError(`测试失败: ${err.message || '网络连接超时'}`);
    } finally {
        testingConnection.value = false;
    }
};

const handleSubmit = async () => {
    if (!formRef.value) return;
    await formRef.value.validate(async (valid) => {
        if (!valid) return;
        submitting.value = true;
        try {
            if (isEdit.value) {
                await updateNode({
                    id: form.id,
                    name: form.name,
                    address: form.address,
                    secretToken: form.secretToken,
                    description: form.description,
                });
                MsgSuccess('节点更新成功');
            } else {
                await createNode({
                    name: form.name,
                    address: form.address,
                    secretToken: form.secretToken,
                    description: form.description,
                });
                MsgSuccess('节点添加成功');
            }
            dialogVisible.value = false;
            loadData();
        } catch (err: any) {
            MsgError(err.message || '提交失败');
        } finally {
            submitting.value = false;
        }
    });
};

const handleDelete = async (id: string) => {
    try {
        await deleteNode(id);
        MsgSuccess('节点删除成功');
        loadData();
    } catch (err: any) {
        MsgError(err.message || '删除失败');
    }
};
</script>

<style scoped lang="scss"></style>
