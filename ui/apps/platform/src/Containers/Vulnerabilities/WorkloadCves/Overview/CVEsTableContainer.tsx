import React, { useState } from 'react';
import { useQuery } from '@apollo/client';
import { Bullseye, Divider, DropdownItem, Spinner, ToolbarItem } from '@patternfly/react-core';

import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useMap from 'hooks/useMap';
import { getHasSearchApplied } from 'utils/searchUtils';
import { VulnerabilityState } from 'types/cve.proto';
import { VulnerabilityException } from 'services/VulnerabilityExceptionService';
import CVEsTable, { cveListQuery, unfilteredImageCountQuery } from '../Tables/CVEsTable';
import TableErrorComponent from '../components/TableErrorComponent';
import { EntityCounts } from '../components/EntityTypeToggleGroup';
import { DefaultFilters, VulnerabilitySeverityLabel } from '../types';
import { getVulnStateScopedQueryString, parseQuerySearchFilter } from '../searchUtils';
import { defaultCVESortFields, CVEsDefaultSort } from '../sortUtils';
import TableEntityToolbar from '../components/TableEntityToolbar';
import ExceptionRequestModal, {
    ExceptionRequestModalOptions,
    ExceptionRequestModalProps,
} from '../components/ExceptionRequestModal/ExceptionRequestModal';
import CompletedExceptionRequestModal from '../components/ExceptionRequestModal/CompletedExceptionRequestModal';

export type CVEsTableContainerProps = {
    defaultFilters: DefaultFilters;
    countsData: EntityCounts;
    vulnerabilityState?: VulnerabilityState; // TODO Make this required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
    pagination: ReturnType<typeof useURLPagination>;
    isUnifiedDeferralsEnabled: boolean; // TODO Remove this when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
};

function CVEsTableContainer({
    defaultFilters,
    countsData,
    vulnerabilityState,
    pagination,
    isUnifiedDeferralsEnabled,
}: CVEsTableContainerProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams, setSortOption } = useURLSort({
        sortFields: defaultCVESortFields,
        defaultSortOption: CVEsDefaultSort,
        onSort: () => setPage(1),
    });

    const { error, loading, data, previousData } = useQuery(cveListQuery, {
        variables: {
            query: getVulnStateScopedQueryString(querySearchFilter, vulnerabilityState),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
        },
    });

    const { data: imageCountData } = useQuery(unfilteredImageCountQuery);

    const selectedCves = useMap<string, ExceptionRequestModalProps['cves'][number]>();
    const [exceptionRequestModalOptions, setExceptionRequestModalOptions] =
        useState<ExceptionRequestModalOptions>(null);

    const [completedException, setCompletedException] = useState<VulnerabilityException | null>(
        null
    );

    function openDeferralModal() {
        setExceptionRequestModalOptions({
            type: 'DEFERRAL',
            cves: Array.from(selectedCves.values()),
        });
    }

    function openFalsePositiveModal() {
        setExceptionRequestModalOptions({
            type: 'FALSE_POSITIVE',
            cves: Array.from(selectedCves.values()),
        });
    }

    const tableData = data ?? previousData;
    return (
        <>
            {exceptionRequestModalOptions && (
                <ExceptionRequestModal
                    cves={exceptionRequestModalOptions.cves}
                    type={exceptionRequestModalOptions.type}
                    scopeContext="GLOBAL"
                    onExceptionRequestSuccess={(vulnerabilityException) => {
                        setExceptionRequestModalOptions(null);
                        selectedCves.clear();
                        setCompletedException(vulnerabilityException);
                    }}
                    onClose={() => setExceptionRequestModalOptions(null)}
                />
            )}
            {completedException && (
                <CompletedExceptionRequestModal
                    exceptionRequest={completedException}
                    onClose={() => setCompletedException(null)}
                />
            )}
            <TableEntityToolbar
                defaultFilters={defaultFilters}
                countsData={countsData}
                setSortOption={setSortOption}
                pagination={pagination}
                tableRowCount={countsData.imageCVECount}
                isFiltered={isFiltered}
            >
                <ToolbarItem alignment={{ default: 'alignRight' }}>
                    <BulkActionsDropdown isDisabled={selectedCves.size === 0}>
                        <DropdownItem
                            key="bulk-defer-cve"
                            component="button"
                            onClick={openDeferralModal}
                        >
                            Defer CVEs
                        </DropdownItem>
                        <DropdownItem
                            key="bulk-mark-false-positive"
                            component="button"
                            onClick={openFalsePositiveModal}
                        >
                            Mark as false positives
                        </DropdownItem>
                    </BulkActionsDropdown>
                </ToolbarItem>
                <ToolbarItem alignment={{ default: 'alignRight' }} variant="separator" />
            </TableEntityToolbar>
            <Divider component="div" />
            {loading && !tableData && (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            )}
            {error && (
                <TableErrorComponent error={error} message="Adjust your filters and try again" />
            )}
            {!error && tableData && (
                <div
                    className="workload-cves-table-container"
                    role="region"
                    aria-live="polite"
                    aria-busy={loading ? 'true' : 'false'}
                >
                    <CVEsTable
                        cves={tableData.imageCVEs}
                        unfilteredImageCount={imageCountData?.imageCount || 0}
                        getSortParams={getSortParams}
                        isFiltered={isFiltered}
                        filteredSeverities={searchFilter.Severity as VulnerabilitySeverityLabel[]}
                        selectedCves={selectedCves}
                        cveTableActionHandler={setExceptionRequestModalOptions}
                        showExceptionMenuItems={
                            isUnifiedDeferralsEnabled && vulnerabilityState === 'OBSERVED'
                        }
                    />
                </div>
            )}
        </>
    );
}

export default CVEsTableContainer;
