<template>
    <div class="node-selector-container">
        <div class="node-selector-inner">
            <el-tag v-if="nodeStore.isRemoteNode" type="warning" effect="dark" class="node-badge" round>
                <el-icon class="mr-1"><Platform /></el-icon>
                <span>当前被控从机: {{ nodeStore.currentNodeInfo.name }}</span>
            </el-tag>
            <el-select
                :model-value="nodeStore.currentNode"
                placeholder="切换目标主机节点"
                class="node-select"
                size="default"
                @change="handleNodeChange"
            >
                <template #prefix>
                    <el-icon><Platform /></el-icon>
                </template>
                <el-option v-for="item in nodeStore.nodeList" :key="item.id" :label="item.name" :value="item.id">
                    <div class="node-option-item">
                        <span
                            class="status-dot"
                            :class="item.status === 'online' ? 'dot-online' : 'dot-offline'"
                        ></span>
                        <span class="node-name">{{ item.name }}</span>
                        <span class="node-addr">{{ item.address }}</span>
                    </div>
                </el-option>
                <template #footer>
                    <div class="node-select-footer" @click="goToNodeManagement">
                        <el-icon><Setting /></el-icon>
                        <span>管理集群节点...</span>
                    </div>
                </template>
            </el-select>
            <el-button
                v-if="nodeStore.isRemoteNode"
                type="primary"
                link
                size="small"
                @click="nodeStore.switchNode('local')"
            >
                切回主控
            </el-button>
        </div>
    </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useNodeStore } from '@/store';
import { Platform, Setting } from '@element-plus/icons-vue';

const router = useRouter();
const nodeStore = useNodeStore();

onMounted(() => {
    nodeStore.fetchNodeList();
});

const handleNodeChange = (val: string) => {
    nodeStore.switchNode(val);
};

const goToNodeManagement = () => {
    router.push({ path: '/hosts/nodes' });
};
</script>

<style scoped lang="scss">
.node-selector-container {
    display: inline-flex;
    align-items: center;
}
.node-selector-inner {
    display: flex;
    align-items: center;
    gap: 8px;
}
.node-badge {
    display: inline-flex;
    align-items: center;
    font-weight: 500;
    padding: 0 10px;
    height: 28px;
}
.node-select {
    width: 250px;
}
.node-option-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    .status-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        display: inline-block;
        flex-shrink: 0;
        &.dot-online {
            background-color: #67c23a;
            box-shadow: 0 0 6px rgba(103, 194, 58, 0.6);
        }
        &.dot-offline {
            background-color: #f56c6c;
        }
    }
    .node-name {
        font-weight: 500;
        flex: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .node-addr {
        color: #909399;
        font-size: 12px;
        margin-left: auto;
    }
}
.node-select-footer {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 8px 0;
    cursor: pointer;
    color: #409eff;
    font-size: 13px;
    border-top: 1px solid var(--el-border-color-lighter, #ebeef5);
    &:hover {
        background-color: var(--el-fill-color-light, #f5f7fa);
    }
}
</style>
