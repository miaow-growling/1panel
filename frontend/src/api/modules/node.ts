import http from '@/api';
import { ResPage, SearchWithPage } from '@/api/interface';

export namespace Node {
    export interface NodeInfo {
        id: string;
        name: string;
        address: string;
        secretToken?: string;
        status: string;
        description?: string;
        createdAt?: string;
        updatedAt?: string;
    }

    export interface NodeCreate {
        name: string;
        address: string;
        secretToken: string;
        description?: string;
    }

    export interface NodeUpdate {
        id: string;
        name: string;
        address: string;
        secretToken?: string;
        description?: string;
    }

    export interface NodeTestReq {
        address: string;
        secretToken: string;
    }
}

export const listNodes = () => {
    return http.get<Node.NodeInfo[]>(`/nodes`);
};

export const searchNodes = (params: SearchWithPage) => {
    return http.post<ResPage<Node.NodeInfo>>(`/nodes/search`, params);
};

export const createNode = (params: Node.NodeCreate) => {
    return http.post(`/nodes`, params);
};

export const updateNode = (params: Node.NodeUpdate) => {
    return http.post(`/nodes/update`, params);
};

export const deleteNode = (id: string) => {
    return http.post(`/nodes/del`, { id });
};

export const testNode = (params: Node.NodeTestReq) => {
    return http.post(`/nodes/test`, params);
};
