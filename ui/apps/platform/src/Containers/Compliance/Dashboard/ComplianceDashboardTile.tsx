import React from 'react';
import { Link } from 'react-router-dom';
import { DocumentNode, gql, useQuery } from '@apollo/client';

import { complianceBasePath } from 'routePaths';

import {
    entityCountNounOrdinaryCase,
    entityNounOrdinaryCasePlural,
} from '../entitiesForCompliance';

const CLUSTERS_COUNT = gql`
    query clustersCount {
        results: complianceClusterCount
    }
`;

const NODES_COUNT = gql`
    query nodesCount {
        results: complianceNodeCount
    }
`;

const NAMESPACES_COUNT = gql`
    query namespacesCount {
        results: complianceNamespaceCount
    }
`;

const DEPLOYMENTS_COUNT = gql`
    query deploymentsCount {
        results: complianceDeploymentCount
    }
`;

// Subset of ComplianceEntityType
type DashboardTileEntityType = 'CLUSTER' | 'DEPLOYMENT' | 'NAMESPACE' | 'NODE';

const queryMap: Record<DashboardTileEntityType, DocumentNode> = {
    CLUSTER: CLUSTERS_COUNT,
    DEPLOYMENT: DEPLOYMENTS_COUNT,
    NAMESPACE: NAMESPACES_COUNT,
    NODE: NODES_COUNT,
};

const entityPathSegmentMap: Record<DashboardTileEntityType, string> = {
    CLUSTER: 'clusters',
    DEPLOYMENT: 'deployments',
    NAMESPACE: 'namespaces',
    NODE: 'nodes',
};

export type ComplianceDashboardTileProps = {
    entityType: DashboardTileEntityType;
};

function ComplianceDashboardTile({ entityType }: ComplianceDashboardTileProps) {
    const QUERY = queryMap[entityType];
    const url = `${complianceBasePath}/${entityPathSegmentMap[entityType]}`;

    const { loading, data, error } = useQuery(QUERY);

    if (loading || error || !data) {
        return (
            <div className="btn btn-base h-10 mr-2">
                <div className="flex flex-col text-center">
                    <div>{entityNounOrdinaryCasePlural[entityType]}</div>
                    <div className="text-sm">{error ? '(not scanned)' : 'scanned'}</div>
                </div>
            </div>
        );
    }

    const count: number = typeof data?.results === 'number' ? data.results : 0;
    return (
        <Link to={url} className="btn btn-base h-10 mr-2 no-underline">
            <div className="flex flex-col text-center whitespace-nowrap">
                <div>{entityCountNounOrdinaryCase(count, entityType)}</div>
                <div className="text-sm">(scanned)</div>
            </div>
        </Link>
    );
}

export default ComplianceDashboardTile;
