import React, { useEffect, useState, ReactElement } from 'react';
import {
    Alert,
    Bullseye,
    Button,
    Flex,
    PageSection,
    Popover,
    Spinner,
    Title,
    Tabs,
    Tab,
    TabTitleText,
    Text,
} from '@patternfly/react-core';
import { OutlinedQuestionCircleIcon } from '@patternfly/react-icons';

import { fetchAlerts, fetchAlertCount } from 'services/AlertsService';
import { CancelledPromiseError } from 'services/cancellationUtils';
import useAnalytics from 'hooks/useAnalytics';
import useEntitiesByIdsCache from 'hooks/useEntitiesByIdsCache';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import { VIOLATION_STATES } from 'constants/violationStates';
import { ENFORCEMENT_ACTIONS } from 'constants/enforcementActions';
import { OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import { onURLSearch } from 'Components/CompoundSearchFilter/utils/utils';
import { FilteredWorkflowView } from 'Components/FilteredWorkflowViewSelector/types';
import { SearchFilter } from 'types/search';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useEffectAfterFirstRender from 'hooks/useEffectAfterFirstRender';
import useURLSort from 'hooks/useURLSort';
import { SortOption } from 'types/table';
import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import useInterval from 'hooks/useInterval';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import FilteredWorkflowViewSelector from 'Components/FilteredWorkflowViewSelector/FilteredWorkflowViewSelector';
import useFilteredWorkflowViewURLState from 'Components/FilteredWorkflowViewSelector/useFilteredWorkflowViewURLState';
import ViolationsTablePanel from './ViolationsTablePanel';
import getTableColumnDescriptors from './violationTableColumnDescriptors';
import { ViolationStateTab, violationStateTabs } from './types';

import './ViolationsTablePage.css';

const tabContentId = 'ViolationsTable';

function getFilteredWorkflowViewSearchFilter(
    filteredWorkflowView: FilteredWorkflowView
): SearchFilter {
    switch (filteredWorkflowView) {
        case 'Applications view':
            return {
                'Platform Component': 'false',
                'Entity Type': 'DEPLOYMENT',
            };
        case 'Platform view':
            return {
                'Platform Component': 'true',
                'Entity Type': 'DEPLOYMENT',
            };
        case 'Full view':
        default:
            return {};
    }
}

const violationPageText: Record<
    FilteredWorkflowView,
    {
        title: string;
        description: string;
    }
> = {
    'Applications view': {
        title: 'User workload violations',
        description: 'Violations affecting user-managed workloads',
    },
    'Platform view': {
        title: 'Platform violations',
        description:
            'Violations affecting workloads used by the OpenShift Platform and layered services',
    },
    'Full view': {
        title: 'All violations',
        description:
            'Violations affecting both user and platform workloads, as well as audit log violations for cluster resources.',
    },
};

function getFilteredWorkflowViewText(filteredWorkflowView: FilteredWorkflowView): {
    title: string;
    description: string;
} {
    return violationPageText[filteredWorkflowView];
}

const violationsPageDescription: Record<ViolationStateTab, string> = {
    ACTIVE: 'Build/Deploy-stage violations for workloads currently in violation, along with unresolved Runtime violations.',
    RESOLVED:
        'Build/Deploy-stage violations for workloads that were removed or modified to be compliant, manually resolved Runtime violations, and violations generated before a policy exclusion was added (all lifecycles)',
    ATTEMPTED:
        'Would-be violations that caused workload deployment attempts to be blocked by the Admission Controller.',
};

function getDescriptionForSelectedViolationState(
    selectedViolationState: ViolationStateTab
): string {
    return violationsPageDescription[selectedViolationState];
}

function ViolationsTablePage(): ReactElement {
    const { analyticsTrack } = useAnalytics();
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isPlatformCveSplitEnabled = isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT');

    const [selectedViolationStateTab, setSelectedViolationStateTab] = useURLStringUnion(
        'violationState',
        violationStateTabs
    );
    const { filteredWorkflowView, setFilteredWorkflowView } = useFilteredWorkflowViewURLState();

    const hasExecutableFilter =
        Object.keys(searchFilter).length &&
        Object.values(searchFilter).some((filter) => filter !== '');

    const [isViewFiltered, setIsViewFiltered] = useState(hasExecutableFilter);

    // Handle changes in the current table page.
    const { page, perPage, setPage, setPerPage } = useURLPagination(50);

    // Handle changes in the currently displayed violations.
    const [isLoadingAlerts, setIsLoadingAlerts] = useState(false);
    const [currentPageAlerts, setCurrentPageAlerts] = useEntitiesByIdsCache();
    const [currentPageAlertsErrorMessage, setCurrentPageAlertsErrorMessage] = useState('');
    const [alertCount, setAlertCount] = useState(0);

    // Flag to signal that the table is in an updating state
    // Note: This would be better replaced with a consistent "polling" state in TBodyUnified
    const [isTableDataUpdating, setIsTableDataUpdating] = useState(true);

    // To handle page/count refreshing.
    const [pollEpoch, setPollEpoch] = useState(0);

    // To handle sort options.
    const columns = getTableColumnDescriptors(filteredWorkflowView);
    const sortFields = columns.flatMap(({ sortField }) => (sortField ? [sortField] : []));

    const defaultSortOption: SortOption = {
        field: 'Violation Time',
        direction: 'desc',
    };
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
    });

    const additionalContextFilter = getFilteredWorkflowViewSearchFilter(filteredWorkflowView);

    const onSearch = (payload: OnSearchPayload) => {
        onURLSearch(searchFilter, setSearchFilter, payload);
    };

    const onChangeFilteredWorkflowView = (value) => {
        setFilteredWorkflowView(value);
        setSearchFilter({});
        setPage(1);
        analyticsTrack({ event: 'Filtered Workflow View Selected', properties: { value } });
    };

    useEffectAfterFirstRender(() => {
        if (hasExecutableFilter && !isViewFiltered) {
            // If the user applies a filter to a previously unfiltered table, return to page 1
            setIsViewFiltered(true);
            setPage(1);
        } else if (!hasExecutableFilter && isViewFiltered) {
            // If the user clears all filters after having previously applied filters, return to page 1
            setIsViewFiltered(false);
            setPage(1);
        }
    }, [hasExecutableFilter, isViewFiltered, setIsViewFiltered, setPage]);

    useEffectAfterFirstRender(() => {
        // Prevent viewing a page beyond the maximum page count
        if (page > Math.ceil(alertCount / perPage)) {
            setPage(1);
        }
    }, [alertCount, perPage, setPage]);

    // We will update the poll epoch after 5 seconds to force a refresh of the alert data
    useInterval(() => {
        setPollEpoch(pollEpoch + 1);
    }, 5000);

    // When any of the deps to this effect change, we want to reload the alerts and count.
    useEffect(() => {
        setIsTableDataUpdating(true);
        const filteredWorkflowFilter = getFilteredWorkflowViewSearchFilter(filteredWorkflowView);
        const alertSearchFilter: SearchFilter = {
            ...searchFilter,
            ...filteredWorkflowFilter,
            'Violation State': selectedViolationStateTab,
        };

        const { request: alertRequest, cancel: cancelAlertRequest } = fetchAlerts({
            alertSearchFilter,
            sortOption,
            page,
            perPage,
        });

        // Get the total count of alerts that match the search request.
        const { request: countRequest, cancel: cancelCountRequest } =
            fetchAlertCount(alertSearchFilter);

        Promise.all([alertRequest, countRequest])
            .then(([alerts, counts]) => {
                setCurrentPageAlerts(alerts);
                setAlertCount(counts);
                setCurrentPageAlertsErrorMessage('');
                setIsLoadingAlerts(false);
                setIsTableDataUpdating(false);
            })
            .catch((error) => {
                if (error instanceof CancelledPromiseError) {
                    return;
                }
                setCurrentPageAlerts([]);
                setAlertCount(0);
                const parsedMessage = getAxiosErrorMessage(error);
                setCurrentPageAlertsErrorMessage(parsedMessage);
                setIsLoadingAlerts(false);
                setIsTableDataUpdating(false);
            });

        return () => {
            cancelAlertRequest();
            cancelCountRequest();
        };
    }, [
        searchFilter,
        page,
        sortOption,
        pollEpoch,
        setCurrentPageAlerts,
        setCurrentPageAlertsErrorMessage,
        setAlertCount,
        perPage,
        selectedViolationStateTab,
        filteredWorkflowView,
    ]);

    // We need to be able to identify which alerts are runtime or attempted, and which are not by id.
    const resolvableAlerts: Set<string> = new Set(
        currentPageAlerts
            .filter(
                (alert) =>
                    alert.lifecycleStage === LIFECYCLE_STAGES.RUNTIME ||
                    alert.state === VIOLATION_STATES.ATTEMPTED
            )
            .map((alert) => alert.id as string)
    );

    const excludableAlerts = currentPageAlerts.filter(
        (alert) =>
            alert.enforcementAction !== ENFORCEMENT_ACTIONS.FAIL_DEPLOYMENT_CREATE_ENFORCEMENT
    );

    const { title, description } = getFilteredWorkflowViewText(filteredWorkflowView);

    return (
        <>
            <PageSection variant="light" id="violations-table">
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    spaceItems={{ default: 'spaceItemsNone' }}
                    className="pf-v5-u-flex-grow-1"
                >
                    <Title headingLevel="h1">
                        {isPlatformCveSplitEnabled ? title : 'Violations'}
                    </Title>
                    {isPlatformCveSplitEnabled && (
                        <Popover
                            aria-label="More information about the current page"
                            bodyContent={description}
                        >
                            <Button title="Page description" variant="plain">
                                <OutlinedQuestionCircleIcon />
                            </Button>
                        </Popover>
                    )}
                </Flex>
            </PageSection>
            <PageSection variant="light" className="pf-v5-u-py-0">
                <Tabs
                    activeKey={selectedViolationStateTab}
                    onSelect={(_e, tab) => {
                        setIsLoadingAlerts(true);
                        setSearchFilter({});
                        setPage(1);
                        setSelectedViolationStateTab(tab);
                    }}
                    aria-label="Violation state tabs"
                >
                    <Tab
                        eventKey="ACTIVE"
                        tabContentId={tabContentId}
                        title={<TabTitleText>Active</TabTitleText>}
                    />
                    <Tab
                        eventKey="RESOLVED"
                        tabContentId={tabContentId}
                        title={<TabTitleText>Resolved</TabTitleText>}
                    />
                    <Tab
                        eventKey="ATTEMPTED"
                        tabContentId={tabContentId}
                        title={<TabTitleText>Attempted</TabTitleText>}
                    />
                </Tabs>
            </PageSection>
            {!isPlatformCveSplitEnabled && (
                <PageSection className="pf-v5-u-py-md" component="div" variant="light">
                    <FilteredWorkflowViewSelector
                        filteredWorkflowView={filteredWorkflowView}
                        onChangeFilteredWorkflowView={onChangeFilteredWorkflowView}
                    />
                </PageSection>
            )}
            {isPlatformCveSplitEnabled && (
                <PageSection variant="light">
                    <Text>
                        {getDescriptionForSelectedViolationState(selectedViolationStateTab)}
                    </Text>
                </PageSection>
            )}
            <PageSection variant="default" id={tabContentId}>
                {isLoadingAlerts && (
                    <Bullseye>
                        <Spinner size="xl" />
                    </Bullseye>
                )}
                {!isLoadingAlerts && currentPageAlertsErrorMessage && (
                    <Bullseye>
                        <Alert
                            variant="danger"
                            title={currentPageAlertsErrorMessage}
                            component="p"
                        />
                    </Bullseye>
                )}
                {!isLoadingAlerts && !currentPageAlertsErrorMessage && (
                    <PageSection variant="light">
                        <ViolationsTablePanel
                            violations={currentPageAlerts}
                            violationsCount={alertCount}
                            currentPage={page}
                            setCurrentPage={setPage}
                            resolvableAlerts={resolvableAlerts}
                            excludableAlerts={excludableAlerts}
                            perPage={perPage}
                            setPerPage={setPerPage}
                            getSortParams={getSortParams}
                            columns={columns}
                            searchFilter={searchFilter}
                            onFilterChange={setSearchFilter}
                            onSearch={onSearch}
                            additionalContextFilter={additionalContextFilter}
                            hasActiveViolations={selectedViolationStateTab === 'ACTIVE'}
                            isTableDataUpdating={isTableDataUpdating}
                        />
                    </PageSection>
                )}
            </PageSection>
        </>
    );
}

export default ViolationsTablePage;
