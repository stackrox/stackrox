import React, { useCallback, useContext, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Divider, PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getComplianceClusterStats } from 'services/ComplianceResultsStatsService';
import { getTableUIState } from 'utils/getTableUIState';

import { CLUSTER_QUERY } from './compliance.coverage.constants';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import { coverageProfileClustersPath } from './compliance.coverage.routes';
import { ComplianceProfilesContext } from './ComplianceProfilesProvider';
import CoveragesPageHeader from './CoveragesPageHeader';
import ProfilesToggleGroup from './ProfilesToggleGroup';
import ProfileClustersTable from './ProfileClustersTable';

function ProfileClustersPage() {
    const { profileName } = useParams();
    const { profileScanStats } = useContext(ComplianceProfilesContext);
    const [currentDatetime, setCurrentDatetime] = useState<Date>(new Date());
    const pagination = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);

    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: [CLUSTER_QUERY],
        defaultSortOption: { field: CLUSTER_QUERY, direction: 'asc' },
        onSort: () => setPage(1),
    });

    const fetchProfileClusters = useCallback(
        () => getComplianceClusterStats(profileName, { sortOption, page, perPage }),
        [page, perPage, profileName, sortOption]
    );
    const { data: profileClusters, loading: isLoading, error } = useRestQuery(fetchProfileClusters);

    const tableState = getTableUIState({
        isLoading,
        data: profileClusters?.scanStats,
        error,
        searchFilter: {},
    });

    useEffect(() => {
        if (profileClusters) {
            setCurrentDatetime(new Date());
        }
    }, [profileClusters]);

    return (
        <>
            <PageTitle title="Compliance coverage - Profile clusters" />
            <CoveragesPageHeader />
            <PageSection>
                <ProfilesToggleGroup
                    profiles={profileScanStats.scanStats}
                    route={coverageProfileClustersPath}
                />
            </PageSection>
            <PageSection variant="default">
                <PageSection variant="light" component="div">
                    <Title headingLevel="h2">Profile results</Title>
                    <Divider />
                    <ProfileClustersTable
                        currentDatetime={currentDatetime}
                        pagination={pagination}
                        profileClustersResultsCount={profileClusters?.totalCount ?? 0}
                        profileName={profileName}
                        tableState={tableState}
                        getSortParams={getSortParams}
                    />
                </PageSection>
            </PageSection>
        </>
    );
}

export default ProfileClustersPage;
