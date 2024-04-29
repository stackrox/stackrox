import React, { ReactNode } from 'react';
import {
    Bullseye,
    Divider,
    Flex,
    Grid,
    GridItem,
    PageSection,
    Pagination,
    pluralize,
    Spinner,
    Split,
    SplitItem,
    Text,
    Title,
} from '@patternfly/react-core';
import { DropdownItem } from '@patternfly/react-core/deprecated';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { gql, useQuery } from '@apollo/client';

import useURLSearch from 'hooks/useURLSearch';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { Pagination as PaginationParam } from 'services/types';
import { getHasSearchApplied } from 'utils/searchUtils';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useMap from 'hooks/useMap';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';

import { DynamicTableLabel } from 'Components/DynamicIcon';
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
    parseWorkloadQuerySearchFilter,
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

    const currentVulnerabilityState = useVulnerabilityState();

    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);
    const { page, perPage, setPage, setPerPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'CVE',
            direction: 'desc',
        },
        onSort: () => setPage(1),
    });

    const { data, previousData, loading, error } = useQuery<
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

    let mainContent: ReactNode | null = null;

    const vulnerabilityData = data ?? previousData;

    const showDeferralUI = isUnifiedDeferralsEnabled && currentVulnerabilityState === 'OBSERVED';
    const canSelectRows = showDeferralUI;

    const createTableActions = showDeferralUI ? createExceptionModalActions : undefined;

    if (error) {
        mainContent = (
            <Bullseye>
                <EmptyStateTemplate
                    headingLevel="h2"
                    title={getAxiosErrorMessage(error)}
                    icon={ExclamationCircleIcon}
                    iconClassName="pf-v5-u-danger-color-100"
                >
                    Adjust your filters and try again
                </EmptyStateTemplate>
            </Bullseye>
        );
    } else if (loading && !vulnerabilityData) {
        mainContent = (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    } else if (vulnerabilityData) {
        const hiddenSeverities = getHiddenSeverities(querySearchFilter);
        const hiddenStatuses = getHiddenStatuses(querySearchFilter);
        const vulnCounter = vulnerabilityData.image.imageCVECountBySeverity;
        const { critical, important, moderate, low } = vulnCounter;
        const totalVulnerabilityCount =
            critical.total + important.total + moderate.total + low.total;

        mainContent = (
            <>
                <div className="pf-v5-u-px-lg pf-v5-u-pb-lg">
                    <Grid hasGutter>
                        <GridItem sm={12} md={6} xl2={4}>
                            <BySeveritySummaryCard
                                title="CVEs by severity"
                                severityCounts={vulnCounter}
                                hiddenSeverities={hiddenSeverities}
                            />
                        </GridItem>
                        <GridItem sm={12} md={6} xl2={4}>
                            <CvesByStatusSummaryCard
                                cveStatusCounts={vulnerabilityData.image.imageCVECountBySeverity}
                                hiddenStatuses={hiddenStatuses}
                                isBusy={loading}
                            />
                        </GridItem>
                    </Grid>
                </div>
                <Divider />
                <div className="pf-v5-u-p-lg">
                    <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                        <SplitItem isFilled>
                            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                <Title headingLevel="h2">
                                    {pluralize(totalVulnerabilityCount, 'result', 'results')} found
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
                                    if (totalVulnerabilityCount < (page - 1) * newPerPage) {
                                        setPage(1);
                                    }
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
                            image={vulnerabilityData.image}
                            getSortParams={getSortParams}
                            isFiltered={isFiltered}
                            selectedCves={selectedCves}
                            canSelectRows={canSelectRows}
                            vulnerabilityState={currentVulnerabilityState}
                            createTableActions={createTableActions}
                        />
                    </div>
                </div>
            </>
        );
    }

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
                <VulnerabilityStateTabs isBox onChange={() => setPage(1)} />
                <div className="pf-v5-u-px-sm pf-v5-u-background-color-100">
                    <WorkloadCveFilterToolbar
                        searchOptions={searchOptions}
                        autocompleteSearchContext={{
                            'Image SHA': imageId,
                        }}
                        onFilterChange={() => setPage(1)}
                    />
                </div>
                <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100">
                    {mainContent}
                </div>
            </PageSection>
        </>
    );
}

export default ImagePageVulnerabilities;
