import { useCallback, useContext, useState } from 'react';
import { Navigate, Route, Routes, useParams } from 'react-router-dom-v5-compat';
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
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import SearchFilterSelectInclusive from 'Components/CompoundSearchFilter/components/SearchFilterSelectInclusive';
import type { OnSearchCallback } from 'Components/CompoundSearchFilter/types';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import PageTitle from 'Components/PageTitle';
import { useBooleanLocalStorage } from 'hooks/useLocalStorage';
import useRestQuery from 'hooks/useRestQuery';
import useURLSearch from 'hooks/useURLSearch';
import { getComplianceProfilesStats } from 'services/ComplianceResultsStatsService';
import type { ComplianceProfileScanStats } from 'services/ComplianceResultsStatsService';
import { defaultChartHeight } from 'utils/chartUtils';

import { coverageProfileChecksPath } from './compliance.coverage.routes';
import { createScanConfigFilter } from './compliance.coverage.utils';
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
import {
    attributeForComplianceCheckStatus,
    clusterSearchFilterConfig,
    profileCheckSearchFilterConfig,
} from '../searchFilterConfig';

const searchFilterConfig = [clusterSearchFilterConfig, profileCheckSearchFilterConfig];

function CoveragesPage() {
    const [isDisclaimerAccepted, setIsDisclaimerAccepted] = useBooleanLocalStorage(
        COMPLIANCE_DISCLAIMER_KEY,
        false
    );
    const { navigateWithScanConfigQuery } = useScanConfigRouter();
    const { profileName } = useParams() as { profileName: string };
    const { isLoading: isLoadingScanConfigProfiles, scanConfigProfilesResponse } =
        useContext(ComplianceProfilesContext);
    const { scanConfigurationsQuery, selectedScanConfigName, setSelectedScanConfigName } =
        useContext(ScanConfigurationsContext);
    const [selectedProfileStats, setSelectedProfileStats] = useState<
        undefined | ComplianceProfileScanStats
    >(undefined);

    const { searchFilter, setSearchFilter } = useURLSearch();

    const fetchProfilesStats = useCallback(async () => {
        setSelectedProfileStats(undefined);
        const response = await getComplianceProfilesStats(
            createScanConfigFilter(selectedScanConfigName)
        );
        if (response) {
            const profileStats = response.scanStats.find(
                (profile) => profile.profileName === profileName
            );
            setSelectedProfileStats(profileStats);
        }
        return response;
    }, [profileName, selectedScanConfigName]);

    const { isLoading: isLoadingProfilesStats, error: profilesStatsError } =
        useRestQuery(fetchProfilesStats);

    function handleProfilesToggleChange(selectedProfile: string) {
        navigateWithScanConfigQuery(coverageProfileChecksPath, { profileName: selectedProfile });
    }

    const onSearch: OnSearchCallback = (payload) => {
        setSearchFilter(updateSearchFilter(searchFilter, payload));
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
            <PageSection hasBodyWrapper={false}>
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
                            className="pf-v6-u-background-color-100"
                            columnGap={{ default: 'columnGapNone' }}
                            direction={{ default: 'column', md: 'row' }}
                            flexWrap={{ default: 'nowrap' }}
                            spaceItems={{ default: 'spaceItemsNone' }}
                        >
                            <FlexItem flex={{ default: 'flex_2' }}>
                                <ProfileDetailsHeader
                                    isLoading={isLoadingScanConfigProfiles}
                                    profileName={profileName}
                                    profileDetails={selectedProfileDetails}
                                />
                            </FlexItem>
                            {(selectedProfileStats ||
                                isLoadingProfilesStats ||
                                profilesStatsError) && (
                                <>
                                    <Divider
                                        orientation={{ default: 'horizontal', md: 'vertical' }}
                                    />
                                    <FlexItem
                                        alignSelf={{ default: 'alignSelfStretch' }}
                                        flex={{ default: 'flex_1' }}
                                        style={{
                                            minWidth: '400px',
                                            minHeight: `${defaultChartHeight}px`,
                                        }}
                                    >
                                        <ProfileStatsWidget
                                            error={profilesStatsError}
                                            isLoading={isLoadingProfilesStats}
                                            profileScanStats={selectedProfileStats}
                                        />
                                    </FlexItem>
                                </>
                            )}
                        </Flex>
                        <Divider component="div" />
                        <PageSection hasBodyWrapper={false} className="pf-v6-u-p-0" component="div">
                            <Toolbar>
                                <ToolbarContent>
                                    <CompoundSearchFilter
                                        config={searchFilterConfig}
                                        defaultEntity="Profile check"
                                        searchFilter={searchFilter}
                                        onSearch={onSearch}
                                    />
                                    <SearchFilterSelectInclusive
                                        attribute={attributeForComplianceCheckStatus}
                                        isSeparate
                                        onSearch={onSearch}
                                        searchFilter={searchFilter}
                                    />
                                    <ToolbarGroup className="pf-v6-u-w-100">
                                        <CompoundSearchFilterLabels
                                            attributesSeparateFromConfig={[
                                                attributeForComplianceCheckStatus,
                                            ]}
                                            config={searchFilterConfig}
                                            onFilterChange={setSearchFilter}
                                            searchFilter={searchFilter}
                                        />
                                    </ToolbarGroup>
                                </ToolbarContent>
                            </Toolbar>
                            <Divider />
                            <Routes>
                                <Route path="checks" element={<ProfileChecksPage />} />
                                <Route path="clusters" element={<ProfileClustersPage />} />
                                <Route path="*" element={<Navigate to="checks" replace />} />
                            </Routes>
                        </PageSection>
                    </>
                )}
            </PageSection>
        </>
    );
}

export default CoveragesPage;
