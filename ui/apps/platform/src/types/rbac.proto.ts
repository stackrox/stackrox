export type K8sRole = {
    id: string;
    name: string;
    namespace: string;
    clusterId: string;
    clusterName: string;
    clusterRole: boolean;
    labels: Record<string, string>;
    annotations: Record<string, string>;
    createdAt: string; // ISO 8601 date string
    rules: PolicyRule[];
};

export type PolicyRule = {
    verbs: string[];
    apiGroups: string[];
    resources: string[];
    nonResourceUrls: string[];
    resourceNames: string[];
};

export type K8sRoleBinding = {
    id: string;
    name: string;
    namespace: string;
    clusterId: string;
    clusterName: string;
    clusterRole: boolean;
    labels: Record<string, string>;
    annotations: Record<string, string>;
    createdAt: string; // ISO 8601 date string
    subjects: Subject[];
    roleId: string;
};

export type Subject = {
    id: string;
    kind: SubjectKind;
    name: string;
    namespace: string;
    clusterId: string;
    clusterName: string;
};

export type SubjectKind = 'UNSET_KIND' | 'SERVICE_ACCOUNT' | 'USER' | 'GROUP';

export type PermissionLevel =
    | 'UNSET'
    | 'NONE'
    | 'DEFAULT'
    | 'ELEVATED_IN_NAMESPACE'
    | 'ELEVATED_CLUSTER_WIDE'
    | 'CLUSTER_ADMIN';
