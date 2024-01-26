import React, { useCallback } from 'react';
import { useParams } from 'react-router-dom';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Divider,
    PageSection,
    Skeleton,
    Spinner,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import { complianceEnhancedCoveragePath } from 'routePaths';
import {
    getSingleClusterCombinedStats,
    getSingleClusterStatsByScanConfig,
} from 'services/ComplianceEnhancedService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ClusterDetailsHeader from './ClusterDetailsHeader';
import ClusterDetailsContent from './ClusterDetailsContent';

function ClusterDetails() {
    const { clusterId } = useParams();

    const clusterStatsListQuery = useCallback(
        () => getSingleClusterCombinedStats(clusterId),
        [clusterId]
    );
    const {
        data: clusterStats,
        loading: isLoadingClusterStats,
        error: clusterStatsError,
    } = useRestQuery(clusterStatsListQuery);

    const scanConfigStats = useCallback(
        () => getSingleClusterStatsByScanConfig(clusterId),
        [clusterId]
    );
    const {
        data: scanStats,
        loading: isLoadingScanStats,
        error: scanStatsError,
    } = useRestQuery(scanConfigStats);

    const hasError = clusterStatsError || scanStatsError;

    const renderClusterNameBreadcrumb = () => {
        if (clusterStatsError) {
            return null;
        }

        return (
            <BreadcrumbItem isActive>
                {isLoadingClusterStats ? (
                    <Skeleton screenreaderText="Loading cluster name" width="150px" />
                ) : (
                    clusterStats?.cluster.clusterName
                )}
            </BreadcrumbItem>
        );
    };

    const renderErrors = () => {
        return (
            <>
                {clusterStatsError && (
                    <Alert
                        variant="warning"
                        title="Unable to fetch cluster details"
                        component="div"
                        isInline
                    >
                        {getAxiosErrorMessage(clusterStatsError)}
                    </Alert>
                )}
                {scanStatsError && (
                    <Alert
                        variant="warning"
                        title="Unable to fetch scan details"
                        component="div"
                        isInline
                    >
                        {getAxiosErrorMessage(scanStatsError)}
                    </Alert>
                )}
            </>
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
            {hasError ? (
                renderErrors()
            ) : (
                <>
                    {clusterStats !== null && (
                        <PageSection variant="light">
                            <ClusterDetailsHeader
                                clusterStats={clusterStats}
                                isLoading={isLoadingClusterStats}
                            />
                        </PageSection>
                    )}
                    <PageSection>
                        {isLoadingScanStats ? (
                            <Bullseye>
                                <Spinner isSVG />
                            </Bullseye>
                        ) : (
                            scanStats &&
                            scanStats.length > 0 && (
                                <ClusterDetailsContent scanRecords={scanStats} />
                            )
                        )}
                    </PageSection>
                </>
            )}
        </>
    );
}

export default ClusterDetails;
