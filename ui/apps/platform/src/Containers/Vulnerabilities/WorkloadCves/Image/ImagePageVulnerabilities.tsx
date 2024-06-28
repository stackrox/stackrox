import React from 'react';
import {
    Divider,
    Flex,
    PageSection,
    Pagination,
    pluralize,
    Split,
    SplitItem,
    Text,
    Title,
} from '@patternfly/react-core';
import { DropdownItem } from '@patternfly/react-core/deprecated';
import { gql, useQuery } from '@apollo/client';
import sum from 'lodash/sum';

import useURLSearch from 'hooks/useURLSearch';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { Pagination as PaginationParam } from 'services/types';
import { getHasSearchApplied } from 'utils/searchUtils';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useMap from 'hooks/useMap';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';

import { DynamicTableLabel } from 'Components/DynamicIcon';
import {
    SummaryCardLayout,
    SummaryCard,
} from 'Containers/Vulnerabilities/components/SummaryCardLayout';
import { getTableUIState } from 'utils/getTableUIState';
import AdvancedFiltersToolbar from 'Containers/Vulnerabilities/components/AdvancedFiltersToolbar';
import {
    imageComponentSearchFilterConfig,
    imageCVESearchFilterConfig,
} from 'Components/CompoundSearchFilter/types';
import { createFilterTracker } from 'Containers/Vulnerabilities/utils/telemetry';
import useAnalytics, { WORKLOAD_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import {
    SearchOption,
    IMAGE_CVE_SEARCH_OPTION,
    COMPONENT_SEARCH_OPTION,
    COMPONENT_SOURCE_SEARCH_OPTION,
} from '../../searchOptions';
import WorkloadCveFilterToolbar from '../components/WorkloadCveFilterToolbar';
import CvesByStatusSummaryCard, {
    ResourceCountByCveSeverityAndStatus,
    resourceCountByCveSeverityAndStatusFragment,
} from '../SummaryCards/CvesByStatusSummaryCard';
import ImageVulnerabilitiesTable, {
    ImageVulnerability,
    imageVulnerabilitiesFragment,
} from '../Tables/ImageVulnerabilitiesTable';
import {
    getHiddenSeverities,
    getHiddenStatuses,
    getStatusesForExceptionCount,
    getVulnStateScopedQueryString,
    parseQuerySearchFilter,
} from '../../utils/searchUtils';
import BySeveritySummaryCard from '../../components/BySeveritySummaryCard';
import { imageMetadataContextFragment, ImageMetadataContext } from '../Tables/table.utils';
import VulnerabilityStateTabs from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import ExceptionRequestModal, {
    ExceptionRequestModalProps,
} from '../../components/ExceptionRequestModal/ExceptionRequestModal';
import CompletedExceptionRequestModal from '../../components/ExceptionRequestModal/CompletedExceptionRequestModal';
import useExceptionRequestModal from '../../hooks/useExceptionRequestModal';

export const imageVulnerabilitiesQuery = gql`
    ${imageMetadataContextFragment}
    ${resourceCountByCveSeverityAndStatusFragment}
    ${imageVulnerabilitiesFragment}
    query getCVEsForImage(
        $id: ID!
        $query: String!
        $pagination: Pagination!
        $statusesForExceptionCount: [String!]
    ) {
        image(id: $id) {
            ...ImageMetadataContext
            imageCVECountBySeverity(query: $query) {
                ...ResourceCountsByCVESeverityAndStatus
            }
            imageVulnerabilities(query: $query, pagination: $pagination) {
                ...ImageVulnerabilityFields
            }
        }
    }
`;

const defaultSortFields = ['CVE', 'CVSS', 'Severity'];

const searchOptions: SearchOption[] = [
    IMAGE_CVE_SEARCH_OPTION,
    COMPONENT_SEARCH_OPTION,
    COMPONENT_SOURCE_SEARCH_OPTION,
];

const searchFilterConfig = {
    'Image CVE': imageCVESearchFilterConfig,
    ImageComponent: imageComponentSearchFilterConfig,
};

export type ImagePageVulnerabilitiesProps = {
    imageId: string;
    imageName: {
        registry: string;
        remote: string;
        tag: string;
    };
    refetchAll: () => void;
    pagination: UseURLPaginationResult;
};

function ImagePageVulnerabilities({
    imageId,
    imageName,
    refetchAll,
    pagination,
}: ImagePageVulnerabilitiesProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isUnifiedDeferralsEnabled = isFeatureFlagEnabled('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL');
    const isAdvancedFiltersEnabled = isFeatureFlagEnabled('ROX_VULN_MGMT_ADVANCED_FILTERS');

    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const currentVulnerabilityState = useVulnerabilityState();

    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const { page, perPage, setPage, setPerPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'Severity',
            direction: 'desc',
        },
        onSort: () => setPage(1, 'replace'),
    });

    // TODO Split metadata, counts, and vulnerabilities into separate queries
    const { data, loading, error } = useQuery<
        {
            image: ImageMetadataContext & {
                imageCVECountBySeverity: ResourceCountByCveSeverityAndStatus;
                imageVulnerabilities: ImageVulnerability[];
            };
        },
        {
            id: string;
            query: string;
            pagination: PaginationParam;
            statusesForExceptionCount: string[];
        }
    >(imageVulnerabilitiesQuery, {
        variables: {
            id: imageId,
            query: getVulnStateScopedQueryString(querySearchFilter, currentVulnerabilityState),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
            statusesForExceptionCount: getStatusesForExceptionCount(currentVulnerabilityState),
        },
    });

    const isFiltered = getHasSearchApplied(querySearchFilter);

    const selectedCves = useMap<string, ExceptionRequestModalProps['cves'][number]>();
    const {
        exceptionRequestModalOptions,
        completedException,
        showModal,
        closeModals,
        createExceptionModalActions,
    } = useExceptionRequestModal();

    const showDeferralUI = isUnifiedDeferralsEnabled && currentVulnerabilityState === 'OBSERVED';
    const canSelectRows = showDeferralUI;

    const createTableActions = showDeferralUI ? createExceptionModalActions : undefined;

    const tableState = getTableUIState({
        isLoading: loading,
        data: data?.image.imageVulnerabilities,
        error,
        searchFilter,
    });

    const hiddenSeverities = getHiddenSeverities(querySearchFilter);
    const hiddenStatuses = getHiddenStatuses(querySearchFilter);
    const severityCounts = data?.image.imageCVECountBySeverity;
    const totalVulnerabilityCount = sum([
        severityCounts?.critical.total ?? 0,
        severityCounts?.important.total ?? 0,
        severityCounts?.moderate.total ?? 0,
        severityCounts?.low.total ?? 0,
    ]);

    return (
        <>
            {exceptionRequestModalOptions && (
                <ExceptionRequestModal
                    cves={exceptionRequestModalOptions.cves}
                    type={exceptionRequestModalOptions.type}
                    scopeContext={{ imageName }}
                    onExceptionRequestSuccess={(exception) => {
                        selectedCves.clear();
                        showModal({ type: 'COMPLETION', exception });
                        return refetchAll();
                    }}
                    onClose={closeModals}
                />
            )}
            {completedException && (
                <CompletedExceptionRequestModal
                    exceptionRequest={completedException}
                    onClose={closeModals}
                />
            )}
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>Review and triage vulnerability data scanned on this image</Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                component="div"
            >
                <VulnerabilityStateTabs
                    isBox
                    onChange={() => {
                        setSearchFilter({});
                        setPage(1, 'replace');
                    }}
                />
                <div className="pf-v5-u-px-sm pf-v5-u-background-color-100">
                    {isAdvancedFiltersEnabled ? (
                        <AdvancedFiltersToolbar
                            className="pf-v5-u-pt-lg pf-v5-u-pb-0"
                            searchFilterConfig={searchFilterConfig}
                            searchFilter={searchFilter}
                            onFilterChange={(newFilter, searchPayload) => {
                                setSearchFilter(newFilter);
                                setPage(1, 'replace');
                                trackAppliedFilter(WORKLOAD_CVE_FILTER_APPLIED, searchPayload);
                            }}
                        />
                    ) : (
                        <WorkloadCveFilterToolbar
                            searchOptions={searchOptions}
                            autocompleteSearchContext={{
                                'Image SHA': imageId,
                            }}
                            onFilterChange={() => setPage(1)}
                        />
                    )}
                </div>
                <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100">
                    <SummaryCardLayout error={error} isLoading={loading}>
                        <SummaryCard
                            data={data?.image}
                            loadingText="Loading image vulnerability summary"
                            renderer={({ data }) => (
                                <BySeveritySummaryCard
                                    title="CVEs by severity"
                                    severityCounts={data.imageCVECountBySeverity}
                                    hiddenSeverities={hiddenSeverities}
                                />
                            )}
                        />
                        <SummaryCard
                            data={data?.image}
                            loadingText="Loading image vulnerability summary"
                            renderer={({ data }) => (
                                <CvesByStatusSummaryCard
                                    cveStatusCounts={data.imageCVECountBySeverity}
                                    hiddenStatuses={hiddenStatuses}
                                    isBusy={loading}
                                />
                            )}
                        />
                    </SummaryCardLayout>
                    <Divider />
                    <div className="pf-v5-u-p-lg">
                        <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                            <SplitItem isFilled>
                                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                    <Title headingLevel="h2">
                                        {pluralize(totalVulnerabilityCount, 'result', 'results')}{' '}
                                        found
                                    </Title>
                                    {isFiltered && <DynamicTableLabel />}
                                </Flex>
                            </SplitItem>
                            {canSelectRows && (
                                <>
                                    <SplitItem>
                                        <BulkActionsDropdown isDisabled={selectedCves.size === 0}>
                                            <DropdownItem
                                                key="bulk-defer-cve"
                                                component="button"
                                                onClick={() =>
                                                    showModal({
                                                        type: 'DEFERRAL',
                                                        cves: Array.from(selectedCves.values()),
                                                    })
                                                }
                                            >
                                                Defer CVEs
                                            </DropdownItem>
                                            <DropdownItem
                                                key="bulk-mark-false-positive"
                                                component="button"
                                                onClick={() =>
                                                    showModal({
                                                        type: 'FALSE_POSITIVE',
                                                        cves: Array.from(selectedCves.values()),
                                                    })
                                                }
                                            >
                                                Mark as false positives
                                            </DropdownItem>
                                        </BulkActionsDropdown>
                                    </SplitItem>
                                    <Divider
                                        className="pf-v5-u-px-lg"
                                        orientation={{ default: 'vertical' }}
                                    />
                                </>
                            )}
                            <SplitItem>
                                <Pagination
                                    itemCount={totalVulnerabilityCount}
                                    page={page}
                                    perPage={perPage}
                                    onSetPage={(_, newPage) => setPage(newPage)}
                                    onPerPageSelect={(_, newPerPage) => {
                                        setPerPage(newPerPage);
                                    }}
                                />
                            </SplitItem>
                        </Split>
                        <div
                            className="workload-cves-table-container"
                            role="region"
                            aria-live="polite"
                            aria-busy={loading ? 'true' : 'false'}
                        >
                            <ImageVulnerabilitiesTable
                                imageMetadata={data?.image}
                                tableState={tableState}
                                getSortParams={getSortParams}
                                isFiltered={isFiltered}
                                selectedCves={selectedCves}
                                canSelectRows={canSelectRows}
                                vulnerabilityState={currentVulnerabilityState}
                                createTableActions={createTableActions}
                                onClearFilters={() => {
                                    setSearchFilter({});
                                    setPage(1, 'replace');
                                }}
                            />
                        </div>
                    </div>
                </div>
            </PageSection>
        </>
    );
}

export default ImagePageVulnerabilities;
