export type NamespaceMetadata = {
    id: string;
    name: string;
    clusterId: string;
    clusterName: string;
    labels: Record<string, string>;
    creationTime: string; // ISO 8601 date string
    priority: number;
    annotations: Record<string, string>;
};
