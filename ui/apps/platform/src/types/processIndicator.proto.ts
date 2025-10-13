// TODO verify if any properties can be optional or have null as value.

export type ProcessIndicator = {
    id: string;
    deploymentId: string;
    containerName: string;
    podId: string;
    podUid: string;
    signal: ProcessSignal;
    clusterId: string;
    namespace: string;
    containerStartTime: string; // ISO 8601 date string
    imageId?: string;
};

export type ProcessSignal = {
    id: string;
    containerId: string;
    time: string | null; // ISO 8601 date string
    name: string;
    args: string;
    execFilePath: string;
    pid: number; // uint32
    uid: number; // uint32
    gid: number; // uint32
    lineage: string[]; // deprecated
    scraped: boolean;
    lineageInfo: LineageInfo[];
};

export type LineageInfo = {
    parentUid: number; // uint32
    parentExecFilePath: string;
};
