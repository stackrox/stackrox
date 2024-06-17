import React, { useCallback, useContext, useEffect, useState } from 'react';
import { generatePath, useHistory, useParams } from 'react-router-dom';
import {
    Divider,
    PageSection,
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

import { CHECK_NAME_QUERY, CLUSTER_QUERY } from './compliance.coverage.constants';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import { coverageProfileClustersPath } from './compliance.coverage.routes';
import { ComplianceProfilesContext } from './ComplianceProfilesProvider';
import ProfileDetailsHeader from './components/ProfileDetailsHeader';
import CoveragesPageHeader from './CoveragesPageHeader';
import ProfilesToggleGroup from './ProfilesToggleGroup';
import ProfileClustersTable from './ProfileClustersTable';

function ProfileClustersPage() {
    const [isDisclaimerAccepted, setIsDisclaimerAccepted] = useBooleanLocalStorage(
        COMPLIANCE_DISCLAIMER_KEY,
        false
    );
    const { profileName } = useParams();
    const history = useHistory();
    const { isLoading: isLoadingScanConfigProfiles, scanConfigProfilesResponse } =
        useContext(ComplianceProfilesContext);
    const [currentDatetime, setCurrentDatetime] = useState<Date>(new Date());
    const pagination = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);

    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: [CLUSTER_QUERY],
        defaultSortOption: { field: CLUSTER_QUERY, direction: 'asc' },
        onSort: () => setPage(1, 'replace'),
    });
    const { searchFilter, setSearchFilter } = useURLSearch();

    const fetchProfileClusters = useCallback(
        () => getComplianceClusterStats(profileName, { sortOption, page, perPage, searchFilter }),
        [page, perPage, profileName, sortOption, searchFilter]
    );
    const { data: profileClusters, loading: isLoading, error } = useRestQuery(fetchProfileClusters);

    const searchFilterConfig = {
        'Profile Check': profileCheckSearchFilterConfig,
        Cluster: getFilteredConfig(clusterSearchFilterConfig, ['Name']),
    };

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

    function handleProfilesToggleChange(selectedProfile: string) {
        const path = generatePath(coverageProfileClustersPath, {
            profileName: selectedProfile,
        });
        history.push(path);
    }

    // @TODO: Consider making a function to make this more reusable
    function onSearch(payload: OnSearchPayload) {
        const { action, category, value } = payload;
        const currentSelection = searchFilter[category] || [];
        let newSelection = !Array.isArray(currentSelection) ? [currentSelection] : currentSelection;
        if (action === 'ADD') {
            newSelection.push(value);
        } else if (action === 'REMOVE') {
            newSelection = newSelection.filter((datum) => datum !== value);
        } else {
            // Do nothing
        }
        setSearchFilter({
            ...searchFilter,
            [category]: newSelection,
        });
    }

    const selectedProfileDetails = scanConfigProfilesResponse?.profiles.find(
        (profile) => profile.name === profileName
    );

    return (
        <>
            <PageTitle title="Compliance coverage - Profile clusters" />
            <CoveragesPageHeader />
            {!isDisclaimerAccepted && (
                <ComplianceUsageDisclaimer onAccept={() => setIsDisclaimerAccepted(true)} />
            )}
            <PageSection>
                <ProfilesToggleGroup
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
                    />
                </PageSection>
            </PageSection>
        </>
    );
}

export default ProfileClustersPage;
