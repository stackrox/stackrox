import React, { ReactNode } from 'react';
import {
    Bullseye,
    Divider,
    DropdownItem,
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
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { gql, useQuery } from '@apollo/client';

import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { Pagination as PaginationParam } from 'services/types';
import { getHasSearchApplied } from 'utils/searchUtils';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useMap from 'hooks/useMap';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import {
    IMAGE_CVE_SEARCH_OPTION,
    SearchOption,
} from 'Containers/Vulnerabilities/components/SearchOptionsDropdown';
import WorkloadTableToolbar from '../components/WorkloadTableToolbar';
import CvesByStatusSummaryCard, {
    ResourceCountByCveSeverityAndStatus,
    resourceCountByCveSeverityAndStatusFragment,
} from '../SummaryCards/CvesByStatusSummaryCard';
import ImageVulnerabilitiesTable, {
    ImageVulnerability,
    imageVulnerabilitiesFragment,
} from '../Tables/ImageVulnerabilitiesTable';
import { DynamicTableLabel } from '../components/DynamicIcon';
import {
    getHiddenSeverities,
    getHiddenStatuses,
    getVulnStateScopedQueryString,
    parseQuerySearchFilter,
} from '../searchUtils';
import BySeveritySummaryCard from '../SummaryCards/BySeveritySummaryCard';
import { imageMetadataContextFragment, ImageMetadataContext } from '../Tables/table.utils';
import VulnerabilityStateTabs from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import ExceptionRequestModal, {
    ExceptionRequestModalProps,
} from '../components/ExceptionRequestModal/ExceptionRequestModal';
import CompletedExceptionRequestModal from '../components/ExceptionRequestModal/CompletedExceptionRequestModal';
import useExceptionRequestModal from '../hooks/useExceptionRequestModal';

const imageVulnerabilitiesQuery = gql`
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

const searchOptions: SearchOption[] = [IMAGE_CVE_SEARCH_OPTION];

export type ImagePageVulnerabilitiesProps = {
    imageId: string;
    imageName: {
        registry: string;
        remote: string;
        tag: string;
    };
};

function ImagePageVulnerabilities({ imageId, imageName }: ImagePageVulnerabilitiesProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isUnifiedDeferralsEnabled = isFeatureFlagEnabled('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL');

    const currentVulnerabilityState = useVulnerabilityState();

    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const { page, perPage, setPage, setPerPage } = useURLPagination(20);
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'CVE',
            direction: 'desc',
        },
        onSort: () => setPage(1),
    });

    const pagination = {
        offset: (page - 1) * perPage,
        limit: perPage,
        sortOption,
    };

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
            pagination,
            statusesForExceptionCount:
                currentVulnerabilityState === 'OBSERVED'
                    ? ['PENDING']
                    : ['APPROVED_PENDING_UPDATE'],
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
                    iconClassName="pf-u-danger-color-100"
                >
                    Adjust your filters and try again
                </EmptyStateTemplate>
            </Bullseye>
        );
    } else if (loading && !vulnerabilityData) {
        mainContent = (
            <Bullseye>
                <Spinner isSVG />
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
                <div className="pf-u-px-lg pf-u-pb-lg">
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
                            />
                        </GridItem>
                    </Grid>
                </div>
                <Divider />
                <div className="pf-u-p-lg">
                    <Split className="pf-u-pb-lg pf-u-align-items-baseline">
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
                                    className="pf-u-px-lg"
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
                    <div className="workload-cves-table-container">
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
            <PageSection component="div" variant="light" className="pf-u-py-md pf-u-px-xl">
                <Text>Review and triage vulnerability data scanned on this image</Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                component="div"
            >
                <VulnerabilityStateTabs isBox />
                <div className="pf-u-px-sm pf-u-background-color-100">
                    <WorkloadTableToolbar
                        searchOptions={searchOptions}
                        autocompleteSearchContext={{
                            'Image SHA': imageId,
                        }}
                        onFilterChange={() => setPage(1)}
                    />
                </div>
                <div className="pf-u-flex-grow-1 pf-u-background-color-100">{mainContent}</div>
            </PageSection>
        </>
    );
}

export default ImagePageVulnerabilities;
