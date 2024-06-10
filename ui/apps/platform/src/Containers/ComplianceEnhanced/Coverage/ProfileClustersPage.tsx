import React, { useCallback, useContext, useEffect, useState } from 'react';
import { generatePath, useHistory, useParams } from 'react-router-dom';
import {
    Divider,
    PageSection,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getComplianceClusterStats } from 'services/ComplianceResultsStatsService';
import { getTableUIState } from 'utils/getTableUIState';

import {
    OnSearchPayload,
    clusterSearchFilterConfig,
    profileCheckSearchFilterConfig,
} from 'Components/CompoundSearchFilter/types';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import { getFilteredConfig } from 'Components/CompoundSearchFilter/utils/searchFilterConfig';
import useURLSearch from 'hooks/useURLSearch';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import { CHECK_NAME_QUERY, CLUSTER_QUERY } from './compliance.coverage.constants';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import { coverageProfileClustersPath } from './compliance.coverage.routes';
import { ComplianceProfilesContext } from './ComplianceProfilesProvider';
import CoveragesPageHeader from './CoveragesPageHeader';
import ProfilesToggleGroup from './ProfilesToggleGroup';
import ProfileClustersTable from './ProfileClustersTable';

function ProfileClustersPage() {
    const { profileName } = useParams();
    const history = useHistory();
    const { profileScanStats } = useContext(ComplianceProfilesContext);
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

    return (
        <>
            <PageTitle title="Compliance coverage - Profile clusters" />
            <CoveragesPageHeader />
            <PageSection>
                <ProfilesToggleGroup
                    profiles={profileScanStats.scanStats}
                    handleToggleChange={handleProfilesToggleChange}
                />
            </PageSection>
            <PageSection variant="default" className="pf-v5-u-py-0">
                <PageSection variant="light" className="pf-v5-u-p-0">
                    <Toolbar>
                        <ToolbarContent>
                            <ToolbarGroup className="pf-v5-u-w-100">
                                <Title headingLevel="h2" className="pf-v5-u-py-md">
                                    Profile results
                                </Title>
                            </ToolbarGroup>
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
