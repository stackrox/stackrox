import type { ReactNode } from 'react';
import {
    Divider,
    DropdownItem,
    Flex,
    PageSection,
    Pagination,
    Split,
    SplitItem,
    Text,
    Title,
    pluralize,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';
import type { DocumentNode } from '@apollo/client';
import type { SearchFilter } from 'types/search';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import type { Pagination as PaginationParam } from 'services/types';
import { getHasSearchApplied, getPaginationParams } from 'utils/searchUtils';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useMap from 'hooks/useMap';
import MenuDropdown from 'Components/PatternFly/MenuDropdown';
import { getSearchFilterConfigWithFeatureFlagDependency } from 'Components/CompoundSearchFilter/utils/utils';
import { DynamicTableLabel } from 'Components/DynamicIcon';
import { getTableUIState } from 'utils/getTableUIState';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import useAnalytics, { WORKLOAD_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import { hideColumnIf, overrideManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import ColumnManagementButton from 'Components/ColumnManagementButton';
import type { VulnerabilityState } from 'types/cve.proto';

import CvesByStatusSummaryCard, {
    resourceCountByCveSeverityAndStatusFragment,
} from '../../components/CvesByStatusSummaryCard';
import type { ResourceCountByCveSeverityAndStatus } from '../../components/CvesByStatusSummaryCard';
import { SummaryCard, SummaryCardLayout } from '../../components/SummaryCardLayout';
import useHasRequestExceptionsAbility from '../../hooks/useHasRequestExceptionsAbility';
import ImageVulnerabilitiesTable, {
    defaultColumns,
    imageVulnerabilitiesFragment,
    tableId,
} from '../Tables/ImageVulnerabilitiesTable';
import type { ImageVulnerability } from '../Tables/ImageVulnerabilitiesTable';
import {
    getHiddenSeverities,
    getHiddenStatuses,
    getStatusesForExceptionCount,
    getVulnStateScopedQueryString,
    parseQuerySearchFilter,
} from '../../utils/searchUtils';
import BySeveritySummaryCard from '../../components/BySeveritySummaryCard';
import {
    imageMetadataContextFragment,
    imageV2MetadataContextFragment,
} from '../Tables/table.utils';
import type { ImageMetadataContext } from '../Tables/table.utils';
import VulnerabilityStateTabs, {
    vulnStateTabContentId,
} from '../components/VulnerabilityStateTabs';
import ExceptionRequestModal from '../../components/ExceptionRequestModal/ExceptionRequestModal';
import type { ExceptionRequestModalProps } from '../../components/ExceptionRequestModal/ExceptionRequestModal';
import CompletedExceptionRequestModal from '../../components/ExceptionRequestModal/CompletedExceptionRequestModal';
import useExceptionRequestModal from '../../hooks/useExceptionRequestModal';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import {
    imageCVESearchFilterConfig,
    imageComponentSearchFilterConfig,
} from '../../searchFilterConfig';
import BaseImageAssessmentCard from '../components/BaseImageAssessmentCard';
import type { BaseImage } from '../components/ImageDetailBadges';

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

export const imageV2VulnerabilitiesQuery = gql`
    ${imageV2MetadataContextFragment}
    ${resourceCountByCveSeverityAndStatusFragment}
    ${imageVulnerabilitiesFragment}
    query getCVEsForImage(
        $id: ID!
        $query: String!
        $pagination: Pagination!
        $statusesForExceptionCount: [String!]
    ) {
        imageV2(id: $id) {
            ...ImageV2MetadataContext
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

export const getImageVulnerabilitiesQuery = (isNewImageDataModelEnabled: boolean): DocumentNode =>
    isNewImageDataModelEnabled ? imageV2VulnerabilitiesQuery : imageVulnerabilitiesQuery;

const defaultSortFields = ['CVE', 'CVSS', 'Severity'];

export type ImagePageVulnerabilitiesProps = {
    imageId: string;
    imageName: {
        registry: string;
        remote: string;
        tag: string;
    };
    baseImage: BaseImage | null;
    refetchAll: () => void;
    pagination: UseURLPaginationResult;
    vulnerabilityState: VulnerabilityState;
    showVulnerabilityStateTabs: boolean;
    additionalToolbarItems?: ReactNode;
    searchFilter: SearchFilter;
    setSearchFilter: (filter: SearchFilter) => void;
};

function ImagePageVulnerabilities({
    imageId,
    imageName,
    baseImage,
    refetchAll,
    pagination,
    vulnerabilityState,
    showVulnerabilityStateTabs,
    additionalToolbarItems,
    searchFilter,
    setSearchFilter,
}: ImagePageVulnerabilitiesProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isBaseImageDetectionEnabled = isFeatureFlagEnabled('ROX_BASE_IMAGE_DETECTION');
    const isNewImageDataModelEnabled = isFeatureFlagEnabled('ROX_FLATTEN_IMAGE_DATA');

    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const { baseSearchFilter } = useWorkloadCveViewContext();

    const hasRequestExceptionsAbility = useHasRequestExceptionsAbility();

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
    const { data, loading, error } = useQuery<
        {
            image:
                | (ImageMetadataContext & {
                      imageCVECountBySeverity: ResourceCountByCveSeverityAndStatus;
                      imageVulnerabilityCount: number;
                      imageVulnerabilities: ImageVulnerability[];
                  })
                | null; // Legacy image data model, will be null when ROX_FLATTEN_IMAGE_DATA is enabled
            imageV2:
                | (ImageMetadataContext & {
                      imageCVECountBySeverity: ResourceCountByCveSeverityAndStatus;
                      imageVulnerabilityCount: number;
                      imageVulnerabilities: ImageVulnerability[];
                  })
                | null; // New image data model, will be null when ROX_FLATTEN_IMAGE_DATA is disabled
        },
        {
            id: string;
            query: string;
            pagination: PaginationParam;
            statusesForExceptionCount: string[];
        }
    >(getImageVulnerabilitiesQuery(isNewImageDataModelEnabled), {
        variables: {
            id: imageId,
            query: getVulnStateScopedQueryString(
                { ...baseSearchFilter, ...querySearchFilter },
                vulnerabilityState
            ),
            pagination: getPaginationParams({ page, perPage, sortOption }),
            statusesForExceptionCount: getStatusesForExceptionCount(vulnerabilityState),
        },
    });

    const imageData =
        (data && (isNewImageDataModelEnabled ? data.imageV2 : data.image)) || undefined;

    const isFiltered = getHasSearchApplied(querySearchFilter);

    const selectedCves = useMap<string, ExceptionRequestModalProps['cves'][number]>();
    const {
        exceptionRequestModalOptions,
        completedException,
        showModal,
        closeModals,
        createExceptionModalActions,
    } = useExceptionRequestModal();

    const showDeferralUI = hasRequestExceptionsAbility && vulnerabilityState === 'OBSERVED';
    const canSelectRows = showDeferralUI;

    const createTableActions = showDeferralUI ? createExceptionModalActions : undefined;

    const tableState = getTableUIState({
        isLoading: loading,
        data: imageData?.imageVulnerabilities,
        error,
        searchFilter,
    });

    const isNvdCvssColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const isEpssProbabilityColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');

    const managedColumnState = useManagedColumns(tableId, defaultColumns);

    const columnConfig = overrideManagedColumns(managedColumnState.columns, {
        cveSelection: hideColumnIf(!canSelectRows),
        nvdCvss: hideColumnIf(!isNvdCvssColumnEnabled),
        epssProbability: hideColumnIf(!isEpssProbabilityColumnEnabled),
        requestDetails: hideColumnIf(vulnerabilityState === 'OBSERVED'),
        rowActions: hideColumnIf(createTableActions === undefined),
    });

    // Keep searchFilterConfigWithFeatureFlagDependency for ROX_SCANNER_V4 also Advisory.
    const searchFilterConfigWithFeatureFlagDependency = [
        // Omit EPSSProbability for 4.7 release until CVE/advisory separation is available in 4.8 release.
        // imageCVESearchFilterConfig,
        {
            ...imageCVESearchFilterConfig,
            attributes: imageCVESearchFilterConfig.attributes.filter(
                ({ searchTerm }) =>
                    searchTerm !== 'EPSS Probability' || isEpssProbabilityColumnEnabled
            ),
        },
        imageComponentSearchFilterConfig,
    ];

    const searchFilterConfig = getSearchFilterConfigWithFeatureFlagDependency(
        isFeatureFlagEnabled,
        searchFilterConfigWithFeatureFlagDependency
    );

    const hiddenSeverities = getHiddenSeverities(querySearchFilter);
    const hiddenStatuses = getHiddenStatuses(querySearchFilter);
    const totalVulnerabilityCount = imageData?.imageVulnerabilityCount ?? 0;

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
            {isBaseImageDetectionEnabled && baseImage && (
                <PageSection component="div" className="pf-v5-u-pt-lg">
                    <BaseImageAssessmentCard baseImage={baseImage} />
                </PageSection>
            )}
            <PageSection
                id={vulnStateTabContentId}
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                component="div"
            >
                {showVulnerabilityStateTabs && (
                    <VulnerabilityStateTabs
                        isBox
                        onChange={() => {
                            setSearchFilter({});
                            setPage(1);
                        }}
                    />
                )}
                <div className="pf-v5-u-px-sm pf-v5-u-background-color-100">
                    <AdvancedFiltersToolbar
                        className="pf-v5-u-pt-lg pf-v5-u-pb-0"
                        searchFilterConfig={searchFilterConfig}
                        defaultSearchFilterEntity="CVE"
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
                    >
                        {additionalToolbarItems}
                    </AdvancedFiltersToolbar>
                </div>
                <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100">
                    <SummaryCardLayout error={error} isLoading={loading}>
                        <SummaryCard
                            data={imageData}
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
                            data={imageData}
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
                                <ColumnManagementButton
                                    columnConfig={columnConfig}
                                    onApplyColumns={managedColumnState.setVisibility}
                                />
                            </SplitItem>
                            {canSelectRows && (
                                <>
                                    <SplitItem>
                                        <MenuDropdown
                                            toggleText="Bulk actions"
                                            isDisabled={selectedCves.size === 0}
                                        >
                                            <DropdownItem
                                                key="bulk-defer-cve"
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
                                                onClick={() =>
                                                    showModal({
                                                        type: 'FALSE_POSITIVE',
                                                        cves: Array.from(selectedCves.values()),
                                                    })
                                                }
                                            >
                                                Mark as false positives
                                            </DropdownItem>
                                        </MenuDropdown>
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
                            style={{ overflowX: 'auto' }}
                            aria-live="polite"
                            aria-busy={loading ? 'true' : 'false'}
                        >
                            <ImageVulnerabilitiesTable
                                imageMetadata={imageData}
                                tableState={tableState}
                                getSortParams={getSortParams}
                                isFiltered={isFiltered}
                                selectedCves={selectedCves}
                                vulnerabilityState={vulnerabilityState}
                                createTableActions={createTableActions}
                                onClearFilters={() => {
                                    setSearchFilter({});
                                    setPage(1);
                                }}
                                tableConfig={columnConfig}
                            />
                        </div>
                    </div>
                </div>
            </PageSection>
        </>
    );
}

export default ImagePageVulnerabilities;
