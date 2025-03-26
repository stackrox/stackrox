import React, { useCallback, useContext } from 'react';
import { useParams } from 'react-router-dom';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Divider,
    Flex,
    Label,
    LabelGroup,
    PageSection,
    Skeleton,
    Spinner,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { onURLSearch } from 'Components/CompoundSearchFilter/utils/utils';
import { OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { getTableUIState } from 'utils/getTableUIState';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getComplianceProfileClusterResults } from 'services/ComplianceResultsService';
import { listComplianceScanConfigClusterProfiles } from 'services/ComplianceScanConfigurationService';
import { addRegexPrefixToFilters } from 'utils/searchUtils';

import ClusterDetailsTable from './ClusterDetailsTable';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import ProfileDetailsHeader from './components/ProfileDetailsHeader';
import { CHECK_NAME_QUERY, CLUSTER_QUERY } from './compliance.coverage.constants';
import {
    coverageProfileClustersPath,
    coverageClusterDetailsPath,
} from './compliance.coverage.routes';
import { createScanConfigFilter, isScanConfigurationDisabled } from './compliance.coverage.utils';
import ScanConfigurationSelect from './components/ScanConfigurationSelect';
import useScanConfigRouter from './hooks/useScanConfigRouter';
import { ScanConfigurationsContext } from './ScanConfigurationsProvider';
import ProfilesToggleGroup from './ProfilesToggleGroup';
import { profileCheckSearchFilterConfig } from '../searchFilterConfig';

const searchFilterConfig = [profileCheckSearchFilterConfig];

function ClusterDetailsPage() {
    const { scanConfigurationsQuery, selectedScanConfigName, setSelectedScanConfigName } =
        useContext(ScanConfigurationsContext);
    const { clusterId, profileName } = useParams() as { clusterId: string; profileName: string };
    const { generatePathWithScanConfig, navigateWithScanConfigQuery } = useScanConfigRouter();
    const pagination = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: [CHECK_NAME_QUERY],
        defaultSortOption: { field: CHECK_NAME_QUERY, direction: 'asc' },
        onSort: () => setPage(1),
    });
    const { searchFilter, setSearchFilter } = useURLSearch();

    const fetchProfilesStats = useCallback(
        () =>
            listComplianceScanConfigClusterProfiles(
                clusterId,
                createScanConfigFilter(selectedScanConfigName)
            ),
        [clusterId, selectedScanConfigName]
    );
    const {
        data: scanConfigProfilesResponse,
        isLoading: isLoadingScanConfigProfiles,
        error: scanConfigProfilesError,
    } = useRestQuery(fetchProfilesStats);

    const fetchCheckResults = useCallback(() => {
        const regexSearchFilter = addRegexPrefixToFilters(searchFilter, [
            CHECK_NAME_QUERY,
            CLUSTER_QUERY,
        ]);
        return getComplianceProfileClusterResults(profileName, clusterId, {
            page,
            perPage,
            sortOption,
            searchFilter: regexSearchFilter,
        });
    }, [clusterId, page, perPage, profileName, sortOption, searchFilter]);
    const {
        data: checkResultsResponse,
        isLoading: isLoadingCheckResults,
        error: checkResultsError,
    } = useRestQuery(fetchCheckResults);

    const tableState = getTableUIState({
        isLoading: isLoadingCheckResults,
        data: checkResultsResponse?.checkResults,
        error: checkResultsError,
        searchFilter,
    });

    function handleProfilesToggleChange(selectedProfile: string) {
        navigateWithScanConfigQuery(coverageClusterDetailsPath, {
            profileName: selectedProfile,
            clusterId,
        });
    }

    const onSearch = (payload: OnSearchPayload) => {
        onURLSearch(searchFilter, setSearchFilter, payload);
    };

    const onCheckStatusSelect = (
        filterType: 'Compliance Check Status',
        checked: boolean,
        selection: string
    ) => {
        const action = checked ? 'ADD' : 'REMOVE';
        const category = filterType;
        const value = selection;
        onSearch({ action, category, value });
    };

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

    if (scanConfigProfilesError) {
        return (
            <Alert
                variant="warning"
                title="Unable to fetch cluster profiles"
                component="p"
                isInline
            >
                {getAxiosErrorMessage(scanConfigProfilesError)}
            </Alert>
        );
    }

    const selectedProfileDetails = scanConfigProfilesResponse?.profiles.find(
        (profile) => profile.name === profileName
    );

    return (
        <>
            <PageTitle title="Compliance coverage - Cluster" />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink
                        to={generatePathWithScanConfig(coverageProfileClustersPath, {
                            profileName,
                        })}
                    >
                        Clusters
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {isLoadingScanConfigProfiles ? (
                            <Skeleton screenreaderText="Loading cluster name" width="150px" />
                        ) : (
                            scanConfigProfilesResponse?.clusterName
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <ScanConfigurationSelect
                isLoading={scanConfigurationsQuery.isLoading}
                scanConfigs={scanConfigurationsQuery.response.configurations}
                selectedScanConfigName={selectedScanConfigName}
                isScanConfigDisabled={(config) =>
                    isScanConfigurationDisabled(config, { clusterId })
                }
                setSelectedScanConfigName={setSelectedScanConfigName}
            />
            <Divider component="div" />
            <PageSection variant="light">
                <Flex
                    direction={{ default: 'column' }}
                    alignItems={{ default: 'alignItemsFlexStart' }}
                >
                    <Title headingLevel="h1" className="pf-v5-u-w-100">
                        {isLoadingScanConfigProfiles ? (
                            <Skeleton fontSize="2xl" screenreaderText="Loading cluster name" />
                        ) : (
                            scanConfigProfilesResponse?.clusterName
                        )}
                    </Title>
                    <LabelGroup numLabels={1}>
                        <Label>
                            {isLoadingScanConfigProfiles ? (
                                <Skeleton
                                    screenreaderText="Loading number of profiles scanned on cluster"
                                    width="135px"
                                />
                            ) : (
                                `Scanned by: ${scanConfigProfilesResponse?.totalCount} profiles`
                            )}
                        </Label>
                    </LabelGroup>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                {isLoadingScanConfigProfiles ? (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                ) : (
                    <>
                        <ProfilesToggleGroup
                            profileName={profileName}
                            profiles={scanConfigProfilesResponse?.profiles ?? []}
                            handleToggleChange={handleProfilesToggleChange}
                        />
                        <Divider component="div" />
                        <ProfileDetailsHeader
                            isLoading={isLoadingScanConfigProfiles}
                            profileName={profileName}
                            profileDetails={selectedProfileDetails}
                        />
                        <Divider component="div" />
                        <ClusterDetailsTable
                            checkResultsCount={checkResultsResponse?.totalCount ?? 0}
                            profileName={profileName}
                            tableState={tableState}
                            pagination={pagination}
                            getSortParams={getSortParams}
                            searchFilterConfig={searchFilterConfig}
                            searchFilter={searchFilter}
                            onFilterChange={setSearchFilter}
                            onSearch={onSearch}
                            onCheckStatusSelect={onCheckStatusSelect}
                            onClearFilters={onClearFilters}
                        />
                    </>
                )}
            </PageSection>
        </>
    );
}

export default ClusterDetailsPage;
