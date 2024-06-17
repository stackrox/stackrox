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
import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getComplianceProfilesClusterStats } from 'services/ComplianceResultsStatsService';
import { getTableUIState } from 'utils/getTableUIState';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getComplianceProfileClusterResults } from 'services/ComplianceResultsService';
import useURLSearch from 'hooks/useURLSearch';
import { getFilteredConfig } from 'Components/CompoundSearchFilter/utils/searchFilterConfig';
import {
    OnSearchPayload,
    profileCheckSearchFilterConfig,
} from 'Components/CompoundSearchFilter/types';

import { onURLSearch } from 'Components/CompoundSearchFilter/utils/utils';
import ClusterDetailsTable from './ClusterDetailsTable';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
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
        () => getComplianceProfilesClusterStats(clusterId),
        [clusterId]
    );
    const {
        data: clusterProfileData,
        loading: isLoadingClusterProfileData,
        error: clusterProfileDataError,
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

    if (clusterProfileDataError) {
        return (
            <Alert
                variant="warning"
                title="Unable to fetch cluster profiles"
                component="div"
                isInline
            >
                {getAxiosErrorMessage(clusterProfileDataError)}
            </Alert>
        );
    }

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
                        {isLoadingClusterProfileData ? (
                            <Skeleton screenreaderText="Loading cluster name" width="150px" />
                        ) : (
                            clusterProfileData?.clusterName
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
                        {isLoadingClusterProfileData ? (
                            <Skeleton fontSize="2xl" screenreaderText="Loading cluster name" />
                        ) : (
                            clusterProfileData?.clusterName
                        )}
                    </Title>
                    <LabelGroup numLabels={1}>
                        <Label>
                            {isLoadingClusterProfileData ? (
                                <Skeleton
                                    screenreaderText="Loading number of profiles scanned on cluster"
                                    width="135px"
                                />
                            ) : (
                                `Scanned by: ${clusterProfileData?.totalCount} profiles`
                            )}
                        </Label>
                    </LabelGroup>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <ProfilesToggleGroup
                    profiles={clusterProfileData?.scanStats ?? []}
                    handleToggleChange={handleProfilesToggleChange}
                />
            </PageSection>
            <PageSection>
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
