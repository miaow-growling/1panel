import { defineStore } from 'pinia';
import piniaPersistConfig from '@/config/pinia-persist';
import { listNodes, Node } from '@/api/modules/node';

export interface NodeState {
    currentNode: string;
    nodeList: Node.NodeInfo[];
    loading: boolean;
}

export const useNodeStore = defineStore({
    id: 'NodeState',
    state: (): NodeState => ({
        currentNode: 'local',
        nodeList: [
            {
                id: 'local',
                name: 'Master 主节点 (当前服务器)',
                address: '127.0.0.1',
                status: 'online',
                description: '主控管理节点',
            },
        ],
        loading: false,
    }),
    getters: {
        currentNodeInfo: (state): Node.NodeInfo => {
            const found = state.nodeList.find((n) => n.id === state.currentNode);
            return (
                found || {
                    id: 'local',
                    name: 'Master 主节点 (当前服务器)',
                    address: '127.0.0.1',
                    status: 'online',
                    description: '主控管理节点',
                }
            );
        },
        isRemoteNode: (state): boolean => {
            return state.currentNode !== 'local' && !!state.currentNode;
        },
    },
    actions: {
        async fetchNodeList() {
            try {
                this.loading = true;
                const res = await listNodes();
                if (res && res.data) {
                    this.nodeList = res.data;
                    // If current node no longer exists in list, reset to local
                    if (!this.nodeList.some((n) => n.id === this.currentNode)) {
                        this.currentNode = 'local';
                    }
                }
            } catch (err) {
                console.error('Failed to fetch node list:', err);
            } finally {
                this.loading = false;
            }
        },
        switchNode(nodeId: string) {
            if (this.currentNode === nodeId) return;
            this.currentNode = nodeId;
            window.location.reload();
        },
    },
    persist: piniaPersistConfig('NodeState'),
});

export default useNodeStore;
