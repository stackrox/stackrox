import React, { useCallback, useContext, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import {
    Bullseye,
    Divider,
    PageSection,
    Spinner,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import ComplianceUsageDisclaimer, {
    COMPLIANCE_DISCLAIMER_KEY,
} from 'Components/ComplianceUsageDisclaimer';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import {
    OnSearchPayload,
    clusterSearchFilterConfig,
    profileCheckSearchFilterConfig,
} from 'Components/CompoundSearchFilter/types';
import { getFilteredConfig } from 'Components/CompoundSearchFilter/utils/searchFilterConfig';
import PageTitle from 'Components/PageTitle';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import { useBooleanLocalStorage } from 'hooks/useLocalStorage';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { getComplianceClusterStats } from 'services/ComplianceResultsStatsService';
import { getTableUIState } from 'utils/getTableUIState';
import { addRegexPrefixToFilters } from 'utils/searchUtils';

import { onURLSearch } from 'Components/CompoundSearchFilter/utils/utils';
import { CHECK_NAME_QUERY, CLUSTER_QUERY } from './compliance.coverage.constants';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import { coverageProfileClustersPath } from './compliance.coverage.routes';
import { combineSearchFilterWithScanConfig } from './compliance.coverage.utils';
import { ComplianceProfilesContext } from './ComplianceProfilesProvider';
import ProfileDetailsHeader from './components/ProfileDetailsHeader';
import ScanConfigurationSelect from './components/ScanConfigurationSelect';
import CoveragesPageHeader from './CoveragesPageHeader';
import useScanConfigRouter from './hooks/useScanConfigRouter';
import ProfilesToggleGroup from './ProfilesToggleGroup';
import ProfileClustersTable from './ProfileClustersTable';
import { ScanConfigurationsContext } from './ScanConfigurationsProvider';

function ProfileClustersPage() {
    const [isDisclaimerAccepted, setIsDisclaimerAccepted] = useBooleanLocalStorage(
        COMPLIANCE_DISCLAIMER_KEY,
        false
    );
    const { navigateWithScanConfigQuery } = useScanConfigRouter();
    const { profileName } = useParams();
    const { isLoading: isLoadingScanConfigProfiles, scanConfigProfilesResponse } =
        useContext(ComplianceProfilesContext);
    const { scanConfigurationsQuery, selectedScanConfigName, setSelectedScanConfigName } =
        useContext(ScanConfigurationsContext);
    const [currentDatetime, setCurrentDatetime] = useState<Date>(new Date());
    const pagination = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);

    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: [CLUSTER_QUERY],
        defaultSortOption: { field: CLUSTER_QUERY, direction: 'asc' },
        onSort: () => setPage(1, 'replace'),
    });
    const { searchFilter, setSearchFilter } = useURLSearch();

    const fetchProfileClusters = useCallback(() => {
        const regexSearchFilter = addRegexPrefixToFilters(searchFilter, [CHECK_NAME_QUERY]);
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
    const { data: profileClusters, loading: isLoading, error } = useRestQuery(fetchProfileClusters);

    const searchFilterConfig = {
        'Profile Check': profileCheckSearchFilterConfig,
        Cluster: getFilteredConfig(clusterSearchFilterConfig, ['Name']),
    };

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

    function handleProfilesToggleChange(selectedProfile: string) {
        navigateWithScanConfigQuery(
            coverageProfileClustersPath,
            { profileName: selectedProfile },
            searchFilter
        );
    }

    const onSearch = (payload: OnSearchPayload) => {
        onURLSearch(searchFilter, setSearchFilter, payload);
    };

    function onClearFilters() {
        setSearchFilter({});
        setPage(1, 'replace');
    }

    const selectedProfileDetails = scanConfigProfilesResponse?.profiles.find(
        (profile) => profile.name === profileName
    );

    return (
        <>
            <PageTitle title="Compliance coverage - Profile clusters" />
            <CoveragesPageHeader />
            <Divider component="div" />
            <ScanConfigurationSelect
                isLoading={scanConfigurationsQuery.isLoading}
                scanConfigs={scanConfigurationsQuery.response.configurations}
                selectedScanConfigName={selectedScanConfigName}
                setSelectedScanConfigName={setSelectedScanConfigName}
            />
            {!isDisclaimerAccepted && (
                <ComplianceUsageDisclaimer onAccept={() => setIsDisclaimerAccepted(true)} />
            )}
            <PageSection>
                {isLoadingScanConfigProfiles ? (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                ) : (
                    <>
                        <ProfilesToggleGroup
                            profileName={profileName}
                            profiles={scanConfigProfilesResponse.profiles}
                            handleToggleChange={handleProfilesToggleChange}
                        />
                        <Divider component="div" />
                        <ProfileDetailsHeader
                            isLoading={isLoadingScanConfigProfiles}
                            profileName={profileName}
                            profileDetails={selectedProfileDetails}
                        />
                        <Divider component="div" />
                        <PageSection variant="light" className="pf-v5-u-p-0" component="div">
                            <Toolbar>
                                <ToolbarContent>
                                    <ToolbarGroup className="pf-v5-u-w-100">
                                        <ToolbarItem className="pf-v5-u-flex-1">
                                            <CompoundSearchFilter
                                                config={searchFilterConfig}
                                                searchFilter={searchFilter}
                                                onSearch={onSearch}
                                            />
                                        </ToolbarItem>
                                    </ToolbarGroup>
                                    <ToolbarGroup className="pf-v5-u-w-100">
                                        <SearchFilterChips
                                            filterChipGroupDescriptors={[
                                                {
                                                    displayName: 'Profile Check',
                                                    searchFilterName: CHECK_NAME_QUERY,
                                                },
                                                {
                                                    displayName: 'Cluster',
                                                    searchFilterName: CLUSTER_QUERY,
                                                },
                                            ]}
                                        />
                                    </ToolbarGroup>
                                </ToolbarContent>
                            </Toolbar>
                            <Divider />
                            <ProfileClustersTable
                                currentDatetime={currentDatetime}
                                pagination={pagination}
                                profileClustersResultsCount={profileClusters?.totalCount ?? 0}
                                profileName={profileName}
                                tableState={tableState}
                                getSortParams={getSortParams}
                                onClearFilters={onClearFilters}
                            />
                        </PageSection>
                    </>
                )}
            </PageSection>
        </>
    );
}

export default ProfileClustersPage;
