import React, { useCallback, useContext, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { getComplianceClusterStats } from 'services/ComplianceResultsStatsService';
import { getTableUIState } from 'utils/getTableUIState';
import { addRegexPrefixToFilters } from 'utils/searchUtils';

import { CHECK_NAME_QUERY, CLUSTER_QUERY } from './compliance.coverage.constants';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import { combineSearchFilterWithScanConfig } from './compliance.coverage.utils';
import ProfileClustersTable from './ProfileClustersTable';
import { ScanConfigurationsContext } from './ScanConfigurationsProvider';

function ProfileClustersPage() {
    const { profileName } = useParams() as { profileName: string };
    const { selectedScanConfigName } = useContext(ScanConfigurationsContext);
    const [currentDatetime, setCurrentDatetime] = useState<Date>(new Date());
    const pagination = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);

    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: [CLUSTER_QUERY],
        defaultSortOption: { field: CLUSTER_QUERY, direction: 'asc' },
        onSort: () => setPage(1),
    });
    const { searchFilter, setSearchFilter } = useURLSearch();

    const fetchProfileClusters = useCallback(() => {
        const regexSearchFilter = addRegexPrefixToFilters(searchFilter, [
            CHECK_NAME_QUERY,
            CLUSTER_QUERY,
        ]);
        const combinedFilter = combineSearchFilterWithScanConfig(
            regexSearchFilter,
            selectedScanConfigName
        );
        return getComplianceClusterStats(profileName, {
            sortOption,
            page,
            perPage,
            searchFilter: combinedFilter,
        });
    }, [page, perPage, profileName, sortOption, searchFilter, selectedScanConfigName]);
    const { data: profileClusters, isLoading, error } = useRestQuery(fetchProfileClusters);

    const tableState = getTableUIState({
        isLoading,
        data: profileClusters?.scanStats,
        error,
        searchFilter,
    });

    useEffect(() => {
        if (profileClusters) {
            setCurrentDatetime(new Date());
        }
    }, [profileClusters]);

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

    return (
        <ProfileClustersTable
            currentDatetime={currentDatetime}
            pagination={pagination}
            profileClustersResultsCount={profileClusters?.totalCount ?? 0}
            profileName={profileName}
            tableState={tableState}
            getSortParams={getSortParams}
            onClearFilters={onClearFilters}
        />
    );
}

export default ProfileClustersPage;
