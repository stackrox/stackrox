import React, { useCallback, useContext } from 'react';
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
import { getComplianceProfileResults } from 'services/ComplianceResultsService';
import {
    OnSearchPayload,
    clusterSearchFilterConfig,
    profileCheckSearchFilterConfig,
} from 'Components/CompoundSearchFilter/types';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import { getFilteredConfig } from 'Components/CompoundSearchFilter/utils/searchFilterConfig';
import useURLSearch from 'hooks/useURLSearch';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import {
    CHECK_NAME_QUERY,
    CHECK_STATUS_QUERY,
    CLUSTER_QUERY,
} from './compliance.coverage.constants';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import { coverageProfileChecksPath } from './compliance.coverage.routes';
import { ComplianceProfilesContext } from './ComplianceProfilesProvider';
import ProfilesToggleGroup from './ProfilesToggleGroup';
import CoveragesPageHeader from './CoveragesPageHeader';
import ProfileChecksTable from './ProfileChecksTable';
import CheckStatusDropdown from './components/CheckStatusDropdown';

function ProfileChecksPage() {
    const { profileName } = useParams();
    const history = useHistory();
    const { profileScanStats } = useContext(ComplianceProfilesContext);
    const pagination = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: [CHECK_NAME_QUERY],
        defaultSortOption: { field: CHECK_NAME_QUERY, direction: 'asc' },
        onSort: () => setPage(1),
    });
    const { searchFilter, setSearchFilter } = useURLSearch();

    const fetchProfileChecks = useCallback(
        () => getComplianceProfileResults(profileName, { sortOption, page, perPage }),
        [page, perPage, profileName, sortOption]
    );
    const { data: profileChecks, loading: isLoading, error } = useRestQuery(fetchProfileChecks);

    const searchFilterConfig = {
        'Profile Check': profileCheckSearchFilterConfig,
        Cluster: getFilteredConfig(clusterSearchFilterConfig, ['Name']),
    };

    // @TODO: Consider making a function to make this more reusable
    const onSearch = (payload: OnSearchPayload) => {
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
    };

    const onSelectCheckStatus = (
        filterType: 'Compliance Check Status',
        checked: boolean,
        selection: string
    ) => {
        const currentSelection = searchFilter[filterType] || [];
        let newSelection = !Array.isArray(currentSelection) ? [currentSelection] : currentSelection;
        if (checked) {
            newSelection.push(selection);
        } else {
            newSelection = newSelection.filter((value) => value !== selection);
        }
        setSearchFilter({
            ...searchFilter,
            [filterType]: newSelection,
        });
    };

    function handleProfilesToggleChange(selectedProfile: string) {
        const path = generatePath(coverageProfileChecksPath, {
            profileName: selectedProfile,
        });
        history.push(path);
    }

    return (
        <>
            <PageTitle title="Compliance coverage - Profile checks" />
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
                                <ToolbarItem align={{ default: 'alignRight' }}>
                                    <CheckStatusDropdown
                                        searchFilter={searchFilter}
                                        onSelect={onSelectCheckStatus}
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
                                        {
                                            displayName: 'Compliance Status',
                                            searchFilterName: CHECK_STATUS_QUERY,
                                        },
                                    ]}
                                />
                            </ToolbarGroup>
                        </ToolbarContent>
                    </Toolbar>
                    <Divider />
                    <ProfileChecksTable
                        isLoading={isLoading}
                        error={error}
                        profileChecksResults={profileChecks?.profileResults ?? []}
                        profileChecksResultsCount={profileChecks?.totalCount ?? 0}
                        profileName={profileName}
                        pagination={pagination}
                        getSortParams={getSortParams}
                    />
                </PageSection>
            </PageSection>
        </>
    );
}

export default ProfileChecksPage;
