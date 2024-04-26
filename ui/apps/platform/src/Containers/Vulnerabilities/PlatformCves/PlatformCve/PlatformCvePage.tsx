import React, { useEffect, useState } from 'react';
import {
    PageSection,
    Breadcrumb,
    Divider,
    BreadcrumbItem,
    Skeleton,
    Alert,
    Grid,
    GridItem,
    Gallery,
    GalleryItem,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';

import { DEFAULT_PAGE_SIZE } from 'Components/Table';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getTableUIState } from 'utils/getTableUIState';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import CvePageHeader, { CveMetadata } from '../../components/CvePageHeader';
import {
    getOverviewPagePath,
    getRegexScopedQueryString,
    parseWorkloadQuerySearchFilter,
} from '../../utils/searchUtils';
import useAffectedClusters from './useAffectedClusters';
import AffectedClustersTable from './AffectedClustersTable';
import usePlatformCveMetadata from './usePlatformCveMetadata';
import ClustersByTypeSummaryCard from './ClustersByTypeSummaryCard';
import AffectedClustersSummaryCard from './AffectedClustersSummaryCard';

const workloadCveOverviewCvePath = getOverviewPagePath('Platform', {
    entityTab: 'CVE',
});

function PlatformCvePage() {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);

    // We need to scope all queries to the *exact* CVE name so that we don't accidentally get
    // data that matches a prefix of the CVE name in the nested fields
    const { cveId } = useParams() as { cveId: string };
    const exactCveIdSearchRegex = `^${cveId}$`;
    const query = getRegexScopedQueryString({
        ...querySearchFilter,
        CVE: [exactCveIdSearchRegex],
    });

    const { page, perPage } = useURLPagination(DEFAULT_PAGE_SIZE);

    const { affectedClustersRequest, clusterData } = useAffectedClusters(query, page, perPage);

    const metadataRequest = usePlatformCveMetadata(cveId, query, page, perPage);
    const cveName = metadataRequest.data?.platformCVE?.cve;

    const tableState = getTableUIState({
        isLoading: affectedClustersRequest.loading,
        error: affectedClustersRequest.error,
        data: clusterData,
        searchFilter: querySearchFilter,
    });

    return (
        <>
            <PageTitle title={`Platform CVEs - PlatformCVE ${cveName}`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewCvePath}>
                        Platform CVEs
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {cveName ?? <Skeleton screenreaderText="Loading CVE name" width="200px" />}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <CvePageHeader data={metadataRequest.data?.platformCVE} />
            </PageSection>
            <Divider component="div" />
            <PageSection className="pf-v5-u-flex-grow-1">
                <div className="pf-v5-u-background-color-100 pf-v5-u-p-lg">
                    {metadataRequest.error && (
                        <Alert
                            title="There was an error loading the summary data for this deployment"
                            isInline
                            variant="danger"
                        >
                            {getAxiosErrorMessage(metadataRequest.error)}
                        </Alert>
                    )}
                    {metadataRequest.loading && (
                        <Gallery
                            hasGutter
                            minWidths={{
                                // Enforce a 1/3 size, taking into account the GridGap
                                default: 'calc(33.3% - var(--pf-v5-l-gallery--m-gutter--GridGap))',
                            }}
                        >
                            <GalleryItem>
                                <Skeleton
                                    style={{ height: '120px' }}
                                    screenreaderText="Loading affected nodes summary"
                                />
                            </GalleryItem>
                            <GalleryItem>
                                <Skeleton
                                    style={{ height: '120px' }}
                                    screenreaderText="Loading affected nodes by CVE severity summary"
                                />
                            </GalleryItem>
                        </Gallery>
                    )}
                    {metadataRequest.data && (
                        <Gallery
                            hasGutter
                            minWidths={{
                                // Enforce a 1/3 size, taking into account the GridGap
                                default: 'calc(33.3% - var(--pf-v5-l-gallery--m-gutter--GridGap))',
                            }}
                        >
                            <GalleryItem>
                                <AffectedClustersSummaryCard
                                    affectedClusterCount={metadataRequest.data.clusterCount}
                                    totalClusterCount={metadataRequest.data.totalClusterCount}
                                />
                            </GalleryItem>
                            <GalleryItem>
                                <ClustersByTypeSummaryCard
                                    clusterCounts={
                                        metadataRequest.data.platformCVE.clusterCountByType
                                    }
                                />
                            </GalleryItem>
                        </Gallery>
                    )}
                </div>
                <Divider component="div" />
                <div className="pf-v5-u-background-color-100 pf-v5-u-flex-grow-1 pf-v5-u-p-lg">
                    <AffectedClustersTable tableState={tableState} />
                </div>
            </PageSection>
        </>
    );
}

export default PlatformCvePage;
