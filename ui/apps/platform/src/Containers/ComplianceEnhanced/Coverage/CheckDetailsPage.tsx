import React, { useCallback, useEffect, useState } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    PageSection,
    Tab,
    Tabs,
    TabsComponent,
} from '@patternfly/react-core';
import { generatePath, useParams } from 'react-router-dom';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { OnSearchPayload, clusterSearchFilterConfig } from 'Components/CompoundSearchFilter/types';
import { getFilteredConfig } from 'Components/CompoundSearchFilter/utils/searchFilterConfig';
import { onURLSearch } from 'Components/CompoundSearchFilter/utils/utils';
import PageTitle from 'Components/PageTitle';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { ComplianceCheckStatus, ComplianceCheckStatusCount } from 'services/ComplianceCommon';
import { getComplianceProfileCheckStats } from 'services/ComplianceResultsStatsService';
import {
    getComplianceProfileCheckDetails,
    getComplianceProfileCheckResult,
} from 'services/ComplianceResultsService';
import { getTableUIState } from 'utils/getTableUIState';

import CheckDetailsTable from './CheckDetailsTable';
import CheckDetailsInfo from './components/CheckDetailsInfo';
import DetailsPageHeader, { PageHeaderLabel } from './components/DetailsPageHeader';
import { coverageProfileChecksPath } from './compliance.coverage.routes';
import { CLUSTER_QUERY } from './compliance.coverage.constants';
import { getClusterResultsStatusObject } from './compliance.coverage.utils';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';

export const DETAILS_TAB = 'Details';
const RESULTS_TAB = 'Results';

export const TAB_NAV_QUERY = 'detailsTab';
const TAB_NAV_VALUES = [RESULTS_TAB, DETAILS_TAB] as const;

function sortCheckStats(a: ComplianceCheckStatusCount, b: ComplianceCheckStatusCount) {
    const order: ComplianceCheckStatus[] = [
        'PASS',
        'FAIL',
        'MANUAL',
        'ERROR',
        'INFO',
        'NOT_APPLICABLE',
        'INCONSISTENT',
    ];
    return order.indexOf(a.status) - order.indexOf(b.status);
}

function CheckDetails() {
    const { checkName, profileName } = useParams();
    const [currentDatetime, setCurrentDatetime] = useState(new Date());
    const pagination = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: [CLUSTER_QUERY],
        defaultSortOption: { field: CLUSTER_QUERY, direction: 'asc' },
        onSort: () => setPage(1, 'replace'),
    });
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [activeTabKey, setActiveTabKey] = useURLStringUnion(TAB_NAV_QUERY, TAB_NAV_VALUES);

    const fetchCheckStats = useCallback(
        () => getComplianceProfileCheckStats(profileName, checkName),
        [profileName, checkName]
    );
    const {
        data: checkStatsResponse,
        loading: isLoadingCheckStats,
        error: checkStatsError,
    } = useRestQuery(fetchCheckStats);

    const fetchCheckDetails = useCallback(
        () => getComplianceProfileCheckDetails(profileName, checkName),
        [profileName, checkName]
    );
    const {
        data: checkDetailsResponse,
        loading: isLoadingCheckDetails,
        error: CheckDetailsError,
    } = useRestQuery(fetchCheckDetails);

    const fetchCheckResults = useCallback(
        () =>
            getComplianceProfileCheckResult(profileName, checkName, {
                page,
                perPage,
                sortOption,
                searchFilter,
            }),
        [page, perPage, checkName, profileName, sortOption, searchFilter]
    );
    const {
        data: checkResultsResponse,
        loading: isLoadingCheckResults,
        error: checkResultsError,
    } = useRestQuery(fetchCheckResults);

    const searchFilterConfig = {
        Cluster: getFilteredConfig(clusterSearchFilterConfig, ['Name']),
    };

    const tableState = getTableUIState({
        isLoading: isLoadingCheckResults,
        data: checkResultsResponse?.checkResults,
        error: checkResultsError,
        searchFilter: {},
    });

    const checkStatsLabels =
        checkStatsResponse?.checkStats
            .sort(sortCheckStats)
            .reduce((acc, checkStat) => {
                const statusObject = getClusterResultsStatusObject(checkStat.status);
                if (statusObject && checkStat.count > 0) {
                    const label: PageHeaderLabel = {
                        text: `${statusObject.statusText}: ${checkStat.count}`,
                        icon: statusObject.icon,
                        color: statusObject.color,
                    };
                    return [...acc, label];
                }
                return acc;
            }, [] as PageHeaderLabel[])
            .filter((component) => component !== null) || [];

    useEffect(() => {
        if (checkResultsResponse) {
            setCurrentDatetime(new Date());
        }
    }, [checkResultsResponse]);

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

    return (
        <>
            <PageTitle title="Compliance coverage - Check" />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItem>Compliance coverage</BreadcrumbItem>
                    <BreadcrumbItemLink
                        to={generatePath(coverageProfileChecksPath, {
                            profileName,
                        })}
                    >
                        {profileName}
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{checkName}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <DetailsPageHeader
                    isLoading={isLoadingCheckStats}
                    name={checkName}
                    labels={checkStatsLabels}
                    summary={checkStatsResponse?.rationale}
                    nameScreenReaderText="Loading profile check details"
                    metadataScreenReaderText="Loading profile check details"
                    error={checkStatsError}
                    errorAlertTitle="Unable to fetch profile check stats"
                />
            </PageSection>
            <Divider component="div" />
            <Tabs
                activeKey={activeTabKey}
                onSelect={(_e, key) => {
                    setActiveTabKey(key);
                }}
                component={TabsComponent.nav}
                className="pf-v5-u-pl-md pf-v5-u-background-color-100 pf-v5-u-flex-shrink-0"
                role="region"
            >
                <Tab eventKey={RESULTS_TAB} title={RESULTS_TAB} />
                <Tab eventKey={DETAILS_TAB} title={DETAILS_TAB} />
            </Tabs>
            <PageSection>
                {activeTabKey === RESULTS_TAB && (
                    <CheckDetailsTable
                        checkResultsCount={checkResultsResponse?.totalCount ?? 0}
                        currentDatetime={currentDatetime}
                        pagination={pagination}
                        profileName={profileName}
                        tableState={tableState}
                        getSortParams={getSortParams}
                        searchFilterConfig={searchFilterConfig}
                        searchFilter={searchFilter}
                        onSearch={onSearch}
                        onCheckStatusSelect={onCheckStatusSelect}
                    />
                )}
                {activeTabKey === DETAILS_TAB && (
                    <PageSection variant="light" component="div">
                        <CheckDetailsInfo
                            checkDetails={checkDetailsResponse}
                            isLoading={isLoadingCheckDetails}
                            error={CheckDetailsError}
                        />
                    </PageSection>
                )}
            </PageSection>
        </>
    );
}

export default CheckDetails;
