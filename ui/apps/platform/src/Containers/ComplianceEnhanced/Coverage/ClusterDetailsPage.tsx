import React, { useCallback } from 'react';
import { generatePath, useHistory, useParams } from 'react-router-dom';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
    Label,
    LabelGroup,
    PageSection,
    Skeleton,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { getFilteredConfig } from 'Components/CompoundSearchFilter/utils/searchFilterConfig';
import { onURLSearch } from 'Components/CompoundSearchFilter/utils/utils';
import {
    OnSearchPayload,
    profileCheckSearchFilterConfig,
} from 'Components/CompoundSearchFilter/types';
import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { getTableUIState } from 'utils/getTableUIState';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getComplianceProfileClusterResults } from 'services/ComplianceResultsService';
import { listComplianceScanConfigClusterProfiles } from 'services/ComplianceScanConfigurationService';

import ClusterDetailsTable from './ClusterDetailsTable';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import ProfileDetailsHeader from './components/ProfileDetailsHeader';
import { CHECK_NAME_QUERY } from './compliance.coverage.constants';
import {
    coverageProfileClustersPath,
    coverageClusterDetailsPath,
} from './compliance.coverage.routes';
import ProfilesToggleGroup from './ProfilesToggleGroup';

function ClusterDetailsPage() {
    const history = useHistory();
    const { clusterId, profileName } = useParams();
    const pagination = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: [CHECK_NAME_QUERY],
        defaultSortOption: { field: CHECK_NAME_QUERY, direction: 'asc' },
        onSort: () => setPage(1, 'replace'),
    });
    const { searchFilter, setSearchFilter } = useURLSearch();

    const fetchProfilesStats = useCallback(
        () => listComplianceScanConfigClusterProfiles(clusterId),
        [clusterId]
    );
    const {
        data: scanConfigProfilesResponse,
        loading: isLoadingScanConfigProfiles,
        error: scanConfigProfilesError,
    } = useRestQuery(fetchProfilesStats);

    const fetchCheckResults = useCallback(
        () =>
            getComplianceProfileClusterResults(profileName, clusterId, {
                page,
                perPage,
                sortOption,
                searchFilter,
            }),
        [clusterId, page, perPage, profileName, sortOption, searchFilter]
    );
    const {
        data: checkResultsResponse,
        loading: isLoadingCheckResults,
        error: checkResultsError,
    } = useRestQuery(fetchCheckResults);

    const searchFilterConfig = {
        'Profile check': getFilteredConfig(profileCheckSearchFilterConfig, ['Name']),
    };

    const tableState = getTableUIState({
        isLoading: isLoadingCheckResults,
        data: checkResultsResponse?.checkResults,
        error: checkResultsError,
        searchFilter: {},
    });

    function handleProfilesToggleChange(selectedProfile: string) {
        const path = generatePath(coverageClusterDetailsPath, {
            profileName: selectedProfile,
            clusterId,
        });
        history.push(path);
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

    if (scanConfigProfilesError) {
        return (
            <Alert
                variant="warning"
                title="Unable to fetch cluster profiles"
                component="div"
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
                    <BreadcrumbItem>Compliance coverage</BreadcrumbItem>
                    <BreadcrumbItemLink
                        to={generatePath(coverageProfileClustersPath, {
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
                    onSearch={onSearch}
                    onCheckStatusSelect={onCheckStatusSelect}
                />
            </PageSection>
        </>
    );
}

export default ClusterDetailsPage;
