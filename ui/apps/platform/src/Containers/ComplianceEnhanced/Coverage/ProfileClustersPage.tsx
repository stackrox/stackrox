import React, { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Divider, PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getComplianceClusterStats } from 'services/ComplianceResultsStatsService';
import { getTableUIState } from 'utils/getTableUIState';

import CoveragesPageHeader from './CoveragesPageHeader';
import CoveragesToggleGroup from './CoveragesToggleGroup';
import ProfileClustersTable from './ProfileClustersTable';

function ProfileClustersPage() {
    const { profileName } = useParams();
    const [currentDatetime, setCurrentDatetime] = useState<Date>(new Date());
    const pagination = useURLPagination(10);

    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: ['Cluster ID'],
        defaultSortOption: { field: 'Cluster ID', direction: 'asc' },
        onSort: () => setPage(1),
    });

    const fetchProfileClusters = useCallback(
        () => getComplianceClusterStats(profileName, sortOption, page, perPage),
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
                <CoveragesToggleGroup tableView="clusters" />
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
