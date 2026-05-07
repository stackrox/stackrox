import { useCallback, useContext, useMemo } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import { Alert } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { getComplianceProfileResults } from 'services/ComplianceResultsService';
import { getTableUIState } from 'utils/getTableUIState';
import { addRegexPrefixToFilters } from 'utils/searchUtils';

import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import { CHECK_NAME_QUERY, CLUSTER_QUERY } from './compliance.coverage.constants';
import { combineSearchFilterWithScanConfig, getStatusCounts } from './compliance.coverage.utils';
import { ComplianceProfilesContext } from './ComplianceProfilesProvider';
import ProfileChecksTable from './ProfileChecksTable';
import { ScanConfigurationsContext } from './ScanConfigurationsProvider';

function ProfileChecksPage() {
    const { profileName } = useParams() as { profileName: string };

    const { selectedScanConfigName } = useContext(ScanConfigurationsContext);
    const { scanConfigProfilesResponse } = useContext(ComplianceProfilesContext);
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

    const selectedProfile = scanConfigProfilesResponse?.profiles.find(
        (profile) => profile.name === profileName
    );
    const isTailoredProfile = selectedProfile?.operatorKind === 'TAILORED_PROFILE';

    const hasInconsistentCheckCounts = useMemo(() => {
        const results = profileChecks?.profileResults;
        if (!results || results.length < 2) {
            return false;
        }
        const totalCounts = results.map((check) => getStatusCounts(check.checkStats).totalCount);
        return new Set(totalCounts).size > 1;
    }, [profileChecks]);

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

    return (
        <>
            {isTailoredProfile && hasInconsistentCheckCounts && (
                <Alert
                    className="pf-v6-u-mb-md"
                    variant="warning"
                    isInline
                    title="Inconsistent check results across clusters"
                    component="p"
                >
                    The total number of check results differs across clusters for this tailored
                    profile. This may indicate that the tailored profile is defined differently in
                    each cluster, for example due to differences in Compliance Operator versions or
                    the underlying base profile.
                </Alert>
            )}
            <ProfileChecksTable
                profileChecksResultsCount={profileChecks?.totalCount ?? 0}
                profileName={profileName}
                pagination={pagination}
                tableState={tableState}
                getSortParams={getSortParams}
                onClearFilters={onClearFilters}
            />
        </>
    );
}

export default ProfileChecksPage;
