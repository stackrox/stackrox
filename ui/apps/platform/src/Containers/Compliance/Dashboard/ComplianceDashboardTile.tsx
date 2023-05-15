import React from 'react';
import { Link } from 'react-router-dom';
import { DocumentNode, gql, useQuery } from '@apollo/client';
import pluralize from 'pluralize';

import { resourceLabels } from 'messages/common';
import { complianceBasePath } from 'routePaths';

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

type ComplianceEntityType = 'CLUSTER' | 'DEPLOYMENT' | 'NAMESPACE' | 'NODE';

const queryMap: Record<ComplianceEntityType, DocumentNode> = {
    CLUSTER: CLUSTERS_COUNT,
    DEPLOYMENT: DEPLOYMENTS_COUNT,
    NAMESPACE: NAMESPACES_COUNT,
    NODE: NODES_COUNT,
};

const entityPathSegmentMap: Record<ComplianceEntityType, string> = {
    CLUSTER: 'clusters',
    DEPLOYMENT: 'deployments',
    NAMESPACE: 'namespaces',
    NODE: 'nodes',
};

export type ComplianceDashboardTileProps = {
    entityType: ComplianceEntityType;
};

function ComplianceDashboardTile({ entityType }: ComplianceDashboardTileProps) {
    const QUERY = queryMap[entityType];
    const url = `${complianceBasePath}/${entityPathSegmentMap[entityType]}`;
    const resourceLabel = resourceLabels[entityType];

    const { loading, data, error } = useQuery(QUERY);

    if (loading || error || !data) {
        return (
            <div className="btn btn-base h-10 mr-2">
                <div className="flex items-center normal-case">
                    {pluralize(resourceLabel)}
                    <br />
                    {error ? '(not scanned)' : 'scanned'}
                </div>
            </div>
        );
    }

    const count: number = typeof data?.results === 'number' ? data.results : 0;
    return (
        <Link to={url} className="btn btn-base h-10 mr-2 no-underline">
            <div className="flex items-center whitespace-nowrap normal-case">
                {`${count} ${pluralize(resourceLabel, count)}`}
                <br />
                (scanned)
            </div>
        </Link>
    );
}

export default ComplianceDashboardTile;
