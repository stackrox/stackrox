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

import useURLSearch from 'hooks/useURLSearch';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { Pagination as PaginationParam } from 'services/types';
import { getHasSearchApplied, getPaginationParams } from 'utils/searchUtils';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useMap from 'hooks/useMap';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import { getSearchFilterConfigWithFeatureFlagDependency } from 'Components/CompoundSearchFilter/utils/utils';
import { DynamicTableLabel } from 'Components/DynamicIcon';
import {
    SummaryCardLayout,
    SummaryCard,
} from 'Containers/Vulnerabilities/components/SummaryCardLayout';
import { getTableUIState } from 'utils/getTableUIState';
import AdvancedFiltersToolbar from 'Containers/Vulnerabilities/components/AdvancedFiltersToolbar';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import useAnalytics, { WORKLOAD_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import useHasRequestExceptionsAbility from 'Containers/Vulnerabilities/hooks/useHasRequestExceptionsAbility';
import {
    convertToFlatImageComponentSearchFilterConfig, // imageComponentSearchFilterConfig
    convertToFlatImageCveSearchFilterConfig, // imageCVESearchFilterConfig
} from 'Containers/Vulnerabilities/searchFilterConfig';
import { filterManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import CvesByStatusSummaryCard, {
    ResourceCountByCveSeverityAndStatus,
    resourceCountByCveSeverityAndStatusFragment,
} from '../SummaryCards/CvesByStatusSummaryCard';
import ImageVulnerabilitiesTable, {
    ImageVulnerability,
    defaultColumns,
    convertToFlatImageVulnerabilitiesFragment, // imageVulnerabilitiesFragment
    tableId,
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
import VulnerabilityStateTabs, {
    vulnStateTabContentId,
} from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import ExceptionRequestModal, {
    ExceptionRequestModalProps,
} from '../../components/ExceptionRequestModal/ExceptionRequestModal';
import CompletedExceptionRequestModal from '../../components/ExceptionRequestModal/CompletedExceptionRequestModal';
import useExceptionRequestModal from '../../hooks/useExceptionRequestModal';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

// After release, replace temporary function
// with imageVulnerabilitiesQuery
// that has unconditional imageVulnerabilitiesFragment.
export function convertToFlatImageVulnerabilitiesQuery(
    isFlattenCveDataEnabled: boolean // ROX_FLATTEN_CVE_DATA
) {
    return gql`
        ${imageMetadataContextFragment}
        ${resourceCountByCveSeverityAndStatusFragment}
        ${convertToFlatImageVulnerabilitiesFragment(isFlattenCveDataEnabled)}
        query getCVEsForImage(
            $id: ID!
            $query: String!
            $pagination: Pagination!
            $statusesForExceptionCount: [String!]
        ) {
            image(id: $id) {
                ...ImageMetadataContext
                imageVulnerabilityCount(query: $query)
                imageCVECountBySeverity(query: $query) {
                    ...ResourceCountsByCVESeverityAndStatus
                }
                imageVulnerabilities(query: $query, pagination: $pagination) {
                    ...ImageVulnerabilityFields
                }
            }
        }
    `;
}

const defaultSortFields = ['CVE', 'CVSS', 'Severity'];

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

    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const { baseSearchFilter } = useWorkloadCveViewContext();

    const currentVulnerabilityState = useVulnerabilityState();
    const hasRequestExceptionsAbility = useHasRequestExceptionsAbility();

    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const { page, perPage, setPage, setPerPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'Severity',
            direction: 'desc',
        },
        onSort: () => setPage(1),
    });

    // TODO Split metadata, counts, and vulnerabilities into separate queries
    const isFlattenCveDataEnabled = isFeatureFlagEnabled('ROX_FLATTEN_CVE_DATA');
    const imageVulnerabilitiesQuery =
        convertToFlatImageVulnerabilitiesQuery(isFlattenCveDataEnabled);
    const { data, loading, error } = useQuery<
        {
            image: ImageMetadataContext & {
                imageCVECountBySeverity: ResourceCountByCveSeverityAndStatus;
                imageVulnerabilityCount: number;
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
            query: getVulnStateScopedQueryString(
                { ...baseSearchFilter, ...querySearchFilter },
                currentVulnerabilityState
            ),
            pagination: getPaginationParams({ page, perPage, sortOption }),
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

    const showDeferralUI = hasRequestExceptionsAbility && currentVulnerabilityState === 'OBSERVED';
    const canSelectRows = showDeferralUI;

    const createTableActions = showDeferralUI ? createExceptionModalActions : undefined;

    const tableState = getTableUIState({
        isLoading: loading,
        data: data?.image.imageVulnerabilities,
        error,
        searchFilter,
    });

    const isNvdCvssColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    // Omit for 4.7 release until CVE/advisory separation is available in 4.8 release.
    // const isEpssProbabilityColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const isEpssProbabilityColumnEnabled = false;
    // totalAdvisories out of scope for MVP
    /*
    const isAdvisoryColumnEnabled =
        isFeatureFlagEnabled('ROX_SCANNER_V4') &&
        isFeatureFlagEnabled('ROX_CVE_ADVISORY_SEPARATION');
    const filteredColumns = filterManagedColumns(
        defaultColumns,
        (key) =>
            (key !== 'nvdCvss' || isNvdCvssColumnEnabled) &&
            (key !== 'epssProbability' || isEpssProbabilityColumnEnabled) &&
            (key !== 'totalAdvisories' || isAdvisoryColumnEnabled)
    );
    */
    const filteredColumns = filterManagedColumns(
        defaultColumns,
        (key) =>
            (key !== 'nvdCvss' || isNvdCvssColumnEnabled) &&
            (key !== 'epssProbability' || isEpssProbabilityColumnEnabled)
    );
    const managedColumnState = useManagedColumns(tableId, filteredColumns);

    // Although we will delete conditional code for EPSS and flatten after release,
    // keep searchFilterConfigWithFeatureFlagDependency for Advisory in the future.
    const imageCVESearchFilterConfig =
        convertToFlatImageCveSearchFilterConfig(isFlattenCveDataEnabled);
    const searchFilterConfigWithFeatureFlagDependency = [
        // Omit EPSSProbability for 4.7 release until CVE/advisory separation is available in 4.8 release.
        // imageCVESearchFilterConfig,
        {
            ...imageCVESearchFilterConfig,
            attributes: imageCVESearchFilterConfig.attributes.filter(
                ({ searchTerm }) => searchTerm !== 'EPSS Probability'
            ),
        },
        convertToFlatImageComponentSearchFilterConfig(isFlattenCveDataEnabled),
    ];

    const searchFilterConfig = getSearchFilterConfigWithFeatureFlagDependency(
        isFeatureFlagEnabled,
        searchFilterConfigWithFeatureFlagDependency
    );

    const hiddenSeverities = getHiddenSeverities(querySearchFilter);
    const hiddenStatuses = getHiddenStatuses(querySearchFilter);
    const totalVulnerabilityCount = data?.image?.imageVulnerabilityCount ?? 0;

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
                id={vulnStateTabContentId}
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                component="div"
            >
                <VulnerabilityStateTabs
                    isBox
                    onChange={() => {
                        setSearchFilter({});
                        setPage(1);
                    }}
                />
                <div className="pf-v5-u-px-sm pf-v5-u-background-color-100">
                    <AdvancedFiltersToolbar
                        className="pf-v5-u-pt-lg pf-v5-u-pb-0"
                        searchFilterConfig={searchFilterConfig}
                        searchFilter={searchFilter}
                        onFilterChange={(newFilter, searchPayload) => {
                            setSearchFilter(newFilter);
                            setPage(1);
                            trackAppliedFilter(WORKLOAD_CVE_FILTER_APPLIED, searchPayload);
                        }}
                        additionalContextFilter={{
                            'Image SHA': imageId,
                            ...baseSearchFilter,
                        }}
                    />
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
                                />
                            )}
                        />
                    </SummaryCardLayout>
                    <Divider />
                    <div className="pf-v5-u-p-lg">
                        <Split hasGutter className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                            <SplitItem isFilled>
                                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                    <Title headingLevel="h2">
                                        {pluralize(totalVulnerabilityCount, 'result', 'results')}{' '}
                                        found
                                    </Title>
                                    {isFiltered && <DynamicTableLabel />}
                                </Flex>
                            </SplitItem>
                            <SplitItem>
                                <ColumnManagementButton managedColumnState={managedColumnState} />
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
                                    setPage(1);
                                }}
                                tableConfig={managedColumnState.columns}
                            />
                        </div>
                    </div>
                </div>
            </PageSection>
        </>
    );
}

export default ImagePageVulnerabilities;
