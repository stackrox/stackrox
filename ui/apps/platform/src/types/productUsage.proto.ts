export type SecuredUnitsUsage = {
    numNodes: number;
    numCpuUnits: number;
};

export type TimeRange = {
    from: string;
    to: string;
};

export type MaxSecuredUnitsUsageResponse = {
    maxNodes: number;
    maxNodesAt: string;
    maxCpuUnitsAt: string;
    maxCpuUnits: number;
};
