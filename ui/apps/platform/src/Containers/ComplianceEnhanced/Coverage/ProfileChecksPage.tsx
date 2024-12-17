import React, { useCallback, useContext } from 'react';
import { useParams } from 'react-router-dom';

import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { getComplianceProfileResults } from 'services/ComplianceResultsService';
import { getTableUIState } from 'utils/getTableUIState';
import { addRegexPrefixToFilters } from 'utils/searchUtils';

import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import { CHECK_NAME_QUERY, CLUSTER_QUERY } from './compliance.coverage.constants';
import { combineSearchFilterWithScanConfig } from './compliance.coverage.utils';
import ProfileChecksTable from './ProfileChecksTable';
import { ScanConfigurationsContext } from './ScanConfigurationsProvider';

function ProfileChecksPage() {
    const { profileName } = useParams() as { profileName: string };

    const { selectedScanConfigName } = useContext(ScanConfigurationsContext);
    const pagination = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: [CHECK_NAME_QUERY],
        defaultSortOption: { field: CHECK_NAME_QUERY, direction: 'asc' },
        onSort: () => setPage(1),
    });
    const { searchFilter, setSearchFilter } = useURLSearch();

    const fetchProfileChecks = useCallback(() => {
        const regexSearchFilter = addRegexPrefixToFilters(searchFilter, [
            CHECK_NAME_QUERY,
            CLUSTER_QUERY,
        ]);
        const combinedFilter = combineSearchFilterWithScanConfig(
            regexSearchFilter,
            selectedScanConfigName
        );
        return getComplianceProfileResults(profileName, {
            sortOption,
            page,
            perPage,
            searchFilter: combinedFilter,
        });
    }, [page, perPage, profileName, sortOption, searchFilter, selectedScanConfigName]);
    const { data: profileChecks, isLoading, error } = useRestQuery(fetchProfileChecks);

    const tableState = getTableUIState({
        isLoading,
        data: profileChecks?.profileResults,
        error,
        searchFilter,
    });

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

    return (
        <ProfileChecksTable
            profileChecksResultsCount={profileChecks?.totalCount ?? 0}
            profileName={profileName}
            pagination={pagination}
            tableState={tableState}
            getSortParams={getSortParams}
            onClearFilters={onClearFilters}
        />
    );
}

export default ProfileChecksPage;
