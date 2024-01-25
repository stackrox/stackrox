import React, { useCallback } from 'react';
import { useParams } from 'react-router-dom';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    PageSection,
    Skeleton,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import { complianceEnhancedCoveragePath } from 'routePaths';
import { getSingleClusterCombinedStats } from 'services/ComplianceEnhancedService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ClusterDetailsHeader from './ClusterDetailsHeader';

function ClusterDetails() {
    const { clusterId } = useParams();

    const listQuery = useCallback(() => getSingleClusterCombinedStats(clusterId), [clusterId]);
    const { data: clusterStats, loading: isLoadingClusterInfo, error } = useRestQuery(listQuery);

    const renderClusterNameBreadcrumb = () => {
        if (error) {
            return null;
        }

        return (
            <BreadcrumbItem isActive>
                {isLoadingClusterInfo ? (
                    <Skeleton screenreaderText="Loading cluster name" width="150px" />
                ) : (
                    clusterStats?.cluster.clusterName
                )}
            </BreadcrumbItem>
        );
    };

    return (
        <>
            <PageTitle title="Compliance Cluster Details" />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={`${complianceEnhancedCoveragePath}?tableView=Clusters`}>
                        Clusters
                    </BreadcrumbItemLink>
                    {renderClusterNameBreadcrumb()}
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            {error || clusterStats === null ? (
                <Alert variant="warning" title="Unable to cluster details" component="div" isInline>
                    {getAxiosErrorMessage(error)}
                </Alert>
            ) : (
                <>
                    <PageSection variant="light">
                        <ClusterDetailsHeader clusterStats={clusterStats} />
                    </PageSection>
                    <Divider component="div" />
                    <PageSection className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1">
                        <div>table here</div>
                    </PageSection>
                </>
            )}
        </>
    );
}

export default ClusterDetails;
