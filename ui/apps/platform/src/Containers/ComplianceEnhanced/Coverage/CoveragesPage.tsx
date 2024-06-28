import React, { useCallback, useContext, useEffect, useState } from 'react';
import { Route, Switch, useParams } from 'react-router-dom';
import {
    Bullseye,
    Divider,
    Flex,
    FlexItem,
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
import useURLSearch from 'hooks/useURLSearch';
import {
    ComplianceProfileScanStats,
    getComplianceProfilesStats,
} from 'services/ComplianceResultsStatsService';

import { onURLSearch } from 'Components/CompoundSearchFilter/utils/utils';
import { CHECK_NAME_QUERY, CLUSTER_QUERY } from './compliance.coverage.constants';
import {
    coverageProfileChecksPath,
    coverageProfileClustersPath,
} from './compliance.coverage.routes';
import { ComplianceProfilesContext } from './ComplianceProfilesProvider';
import ProfileDetailsHeader from './components/ProfileDetailsHeader';
import ProfileStatsWidget from './components/ProfileStatsWidget';
import ScanConfigurationSelect from './components/ScanConfigurationSelect';
import CoveragesPageHeader from './CoveragesPageHeader';
import useScanConfigRouter from './hooks/useScanConfigRouter';
import ProfilesToggleGroup from './ProfilesToggleGroup';
import ProfileChecksPage from './ProfileChecksPage';
import ProfileClustersPage from './ProfileClustersPage';
import { ScanConfigurationsContext } from './ScanConfigurationsProvider';

function CoveragesPage() {
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
    const [selectedProfileStats, setSelectedProfileStats] = useState<
        undefined | ComplianceProfileScanStats
    >(undefined);

    const { searchFilter, setSearchFilter } = useURLSearch();

    const searchFilterConfig = {
        'Profile Check': profileCheckSearchFilterConfig,
        Cluster: getFilteredConfig(clusterSearchFilterConfig, ['Name']),
    };

    const fetchProfilesStats = useCallback(() => getComplianceProfilesStats(), []);
    const {
        data: profilesStatsResponse,
        loading: isLoadingProfilesStats,
        error: profilesStatsError,
        refetch: refetchProfilesStats,
    } = useRestQuery(fetchProfilesStats);

    // Refetch profiles stats when profileName changes to stay more in sync with the data in the table
    useEffect(() => {
        setSelectedProfileStats(undefined);
        refetchProfilesStats();
    }, [profileName, refetchProfilesStats]);

    useEffect(() => {
        if (profilesStatsResponse) {
            const profileStats = profilesStatsResponse.scanStats.find(
                (profile) => profile.profileName === profileName
            );
            setSelectedProfileStats(profileStats);
        }
    }, [profilesStatsResponse, profileName]);

    function handleProfilesToggleChange(selectedProfile: string) {
        navigateWithScanConfigQuery(coverageProfileChecksPath, { profileName: selectedProfile });
    }

    const onSearch = (payload: OnSearchPayload) => {
        onURLSearch(searchFilter, setSearchFilter, payload);
    };

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
                        <Flex
                            alignItems={{ default: 'alignItemsStretch' }}
                            className="pf-v5-u-background-color-100"
                            columnGap={{ default: 'columnGapNone' }}
                            direction={{ default: 'column', md: 'row' }}
                            flexWrap={{ default: 'nowrap' }}
                            spaceItems={{ default: 'spaceItemsNone' }}
                            style={{ minHeight: '260px' }}
                        >
                            <FlexItem flex={{ default: 'flex_2' }}>
                                <ProfileDetailsHeader
                                    isLoading={isLoadingScanConfigProfiles}
                                    profileName={profileName}
                                    profileDetails={selectedProfileDetails}
                                />
                            </FlexItem>
                            <Divider orientation={{ default: 'horizontal', md: 'vertical' }} />
                            <FlexItem
                                alignSelf={{ default: 'alignSelfStretch' }}
                                flex={{ default: 'flex_1' }}
                                style={{ minWidth: '400px' }}
                            >
                                <ProfileStatsWidget
                                    error={profilesStatsError}
                                    isLoading={isLoadingProfilesStats}
                                    profileScanStats={selectedProfileStats}
                                />
                            </FlexItem>
                        </Flex>
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
                            <Switch>
                                <Route
                                    exact
                                    path={coverageProfileChecksPath}
                                    render={() => <ProfileChecksPage />}
                                />
                                <Route
                                    exact
                                    path={coverageProfileClustersPath}
                                    render={() => <ProfileClustersPage />}
                                />
                            </Switch>
                        </PageSection>
                    </>
                )}
            </PageSection>
        </>
    );
}

export default CoveragesPage;
