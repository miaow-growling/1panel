import http from '@/api';
import { TimeoutEnum } from '@/enums/http-enum';

export const listHostNodes = () => {
    return http.get<any>(`/core/nodes`, {}, { timeout: TimeoutEnum.T_10S });
};

export const createHostNode = (params: any) => {
    return http.post<any>(`/core/nodes`, params, { timeout: TimeoutEnum.T_10S });
};

export const updateHostNode = (params: any) => {
    return http.post<any>(`/core/nodes/update`, params, { timeout: TimeoutEnum.T_10S });
};

export const deleteHostNode = (id: number) => {
    return http.post<any>(`/core/nodes/del`, { id }, { timeout: TimeoutEnum.T_10S });
};

export const testHostNodeByID = (params: any) => {
    return http.post<any>(`/core/nodes/test/id`, params, { timeout: TimeoutEnum.T_10S });
};

export const testHostNode = (params: any) => {
    return http.post<any>(`/core/nodes/test`, params, { timeout: TimeoutEnum.T_10S });
};

export const getInstallScript = (params: any) => {
    return http.post<any>(`/core/nodes/install-script`, params, { timeout: TimeoutEnum.T_10S });
};
