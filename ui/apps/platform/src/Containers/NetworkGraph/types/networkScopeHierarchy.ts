import { SearchFilter } from 'types/search';

export type NetworkScopeHierarchy = {
    cluster: {
        id: string;
        name: string;
    };
    namespaces: string[];
    deployments: string[];
    remainingQuery: Omit<SearchFilter, 'Cluster' | 'Namespace' | 'Deployment'>;
};
