import entityTypes from 'constants/entityTypes';

// TODO: Maybe we can eventually pull this to it's own types file where all the data structures can live
export type ClusterNode = {
    classes: string;
    data: {
        id: string;
        name: string;
        active: boolean;
        type: string;
    };
};

/**
 * Create the cluster node for the network graph
 *
 */
export const getClusterNode = (clusterName: string): ClusterNode => {
    const clusterNode = {
        classes: 'cluster',
        data: {
            id: clusterName,
            name: clusterName,
            active: false,
            type: entityTypes.CLUSTER,
        },
    };
    return clusterNode;
};
