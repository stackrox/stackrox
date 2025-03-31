import React, { useCallback, useContext, useEffect, useState } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    PageSection,
    Tab,
    Tabs,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { CompoundSearchFilterConfig, OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import { onURLSearch } from 'Components/CompoundSearchFilter/utils/utils';
import PageTitle from 'Components/PageTitle';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { getComplianceProfileCheckStats } from 'services/ComplianceResultsStatsService';
import {
    getComplianceProfileCheckDetails,
    getComplianceProfileCheckResult,
} from 'services/ComplianceResultsService';
import { getTableUIState } from 'utils/getTableUIState';
import { addRegexPrefixToFilters } from 'utils/searchUtils';

import { Name } from 'Components/CompoundSearchFilter/attributes/cluster';
import CheckDetailsHeader from './CheckDetailsHeader';
import CheckDetailsTable, { tabContentIdForResults } from './CheckDetailsTable';
import {
    combineSearchFilterWithScanConfig,
    createScanConfigFilter,
    isScanConfigurationDisabled,
} from './compliance.coverage.utils';
import CheckDetailsInfo from './components/CheckDetailsInfo';
import { coverageProfileChecksPath } from './compliance.coverage.routes';
import { CLUSTER_QUERY } from './compliance.coverage.constants';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import ScanConfigurationSelect from './components/ScanConfigurationSelect';
import useScanConfigRouter from './hooks/useScanConfigRouter';
import { ScanConfigurationsContext } from './ScanConfigurationsProvider';

export const DETAILS_TAB = 'Details';
const RESULTS_TAB = 'Results';

const tabContentIdForDetails = 'check-details-Details-tab-section';

export const TAB_NAV_QUERY = 'detailsTab';
const TAB_NAV_VALUES = [RESULTS_TAB, DETAILS_TAB] as const;

const searchFilterConfig: CompoundSearchFilterConfig = [
    {
        displayName: 'Cluster',
        searchCategory: 'CLUSTERS',
        attributes: [Name],
    },
];

function CheckDetails() {
    const { scanConfigurationsQuery, selectedScanConfigName, setSelectedScanConfigName } =
        useContext(ScanConfigurationsContext);
    const { checkName, profileName } = useParams() as { checkName: string; profileName: string };
    const { generatePathWithScanConfig } = useScanConfigRouter();
    const [currentDatetime, setCurrentDatetime] = useState(new Date());
    const pagination = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: [CLUSTER_QUERY],
        defaultSortOption: { field: CLUSTER_QUERY, direction: 'asc' },
        onSort: () => setPage(1),
    });
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [activeTabKey, setActiveTabKey] = useURLStringUnion(TAB_NAV_QUERY, TAB_NAV_VALUES);

    const fetchCheckStats = useCallback(
        () =>
            getComplianceProfileCheckStats(
                profileName,
                checkName,
                createScanConfigFilter(selectedScanConfigName)
            ),
        [profileName, checkName, selectedScanConfigName]
    );
    const {
        data: checkStatsResponse,
        isLoading: isLoadingCheckStats,
        error: checkStatsError,
    } = useRestQuery(fetchCheckStats);

    const fetchCheckDetails = useCallback(
        () => getComplianceProfileCheckDetails(profileName, checkName),
        [profileName, checkName]
    );
    const {
        data: checkDetailsResponse,
        isLoading: isLoadingCheckDetails,
        error: CheckDetailsError,
    } = useRestQuery(fetchCheckDetails);

    const fetchCheckResults = useCallback(() => {
        const regexSearchFilter = addRegexPrefixToFilters(searchFilter, [CLUSTER_QUERY]);
        const combinedFilter = combineSearchFilterWithScanConfig(
            regexSearchFilter,
            selectedScanConfigName
        );
        return getComplianceProfileCheckResult(profileName, checkName, {
            page,
            perPage,
            sortOption,
            searchFilter: combinedFilter,
        });
    }, [page, perPage, checkName, profileName, sortOption, searchFilter, selectedScanConfigName]);
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

    useEffect(() => {
        if (checkResultsResponse) {
            setCurrentDatetime(new Date());
        }
    }, [checkResultsResponse]);

    const onSearch = (payload: OnSearchPayload) => {
        onURLSearch(searchFilter, setSearchFilter, payload);
    };

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

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
                    <BreadcrumbItemLink
                        to={generatePathWithScanConfig(coverageProfileChecksPath, {
                            profileName,
                        })}
                    >
                        {profileName}
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{checkName}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <ScanConfigurationSelect
                isLoading={scanConfigurationsQuery.isLoading}
                scanConfigs={scanConfigurationsQuery.response.configurations}
                selectedScanConfigName={selectedScanConfigName}
                isScanConfigDisabled={(config) =>
                    isScanConfigurationDisabled(config, { profileName })
                }
                setSelectedScanConfigName={setSelectedScanConfigName}
            />
            <Divider component="div" />
            <PageSection variant="light">
                <CheckDetailsHeader
                    checkName={checkName}
                    checkStatsResponse={checkStatsResponse}
                    isLoading={isLoadingCheckStats}
                    error={checkStatsError}
                />
            </PageSection>
            <Divider component="div" />
            <Tabs
                activeKey={activeTabKey}
                onSelect={(_e, key) => {
                    setActiveTabKey(key);
                }}
                className="pf-v5-u-pl-md pf-v5-u-background-color-100 pf-v5-u-flex-shrink-0"
            >
                <Tab
                    eventKey={RESULTS_TAB}
                    title={RESULTS_TAB}
                    tabContentId={tabContentIdForResults}
                />
                <Tab
                    eventKey={DETAILS_TAB}
                    title={DETAILS_TAB}
                    tabContentId={tabContentIdForDetails}
                />
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
                        onFilterChange={setSearchFilter}
                        onSearch={onSearch}
                        onCheckStatusSelect={onCheckStatusSelect}
                        onClearFilters={onClearFilters}
                    />
                )}
                {activeTabKey === DETAILS_TAB && (
                    <PageSection variant="light" component="div" id={tabContentIdForDetails}>
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
