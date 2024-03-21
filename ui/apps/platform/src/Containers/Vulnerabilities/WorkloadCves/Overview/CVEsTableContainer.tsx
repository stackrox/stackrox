import React from 'react';
import { useQuery } from '@apollo/client';
import { Bullseye, Divider, Spinner, ToolbarItem } from '@patternfly/react-core';
import { DropdownItem } from '@patternfly/react-core/deprecated';

import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useMap from 'hooks/useMap';
import { getHasSearchApplied } from 'utils/searchUtils';
import { VulnerabilityState } from 'types/cve.proto';

import useInvalidateVulnerabilityQueries from '../../hooks/useInvalidateVulnerabilityQueries';
import CVEsTable, { cveListQuery, unfilteredImageCountQuery } from '../Tables/CVEsTable';
import { VulnerabilitySeverityLabel } from '../../types';
import {
    getStatusesForExceptionCount,
    getVulnStateScopedQueryString,
    parseWorkloadQuerySearchFilter,
} from '../../utils/searchUtils';
import TableEntityToolbar, { TableEntityToolbarProps } from '../../components/TableEntityToolbar';
import ExceptionRequestModal, {
    ExceptionRequestModalProps,
} from '../../components/ExceptionRequestModal/ExceptionRequestModal';
import CompletedExceptionRequestModal from '../../components/ExceptionRequestModal/CompletedExceptionRequestModal';
import useExceptionRequestModal from '../../hooks/useExceptionRequestModal';

export type CVEsTableContainerProps = {
    filterToolbar: TableEntityToolbarProps['filterToolbar'];
    entityToggleGroup: TableEntityToolbarProps['entityToggleGroup'];
    rowCount: number;
    vulnerabilityState?: VulnerabilityState; // TODO Make this required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
    pagination: ReturnType<typeof useURLPagination>;
    sort: ReturnType<typeof useURLSort>;
    isUnifiedDeferralsEnabled: boolean; // TODO Remove this when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
};

function CVEsTableContainer({
    filterToolbar,
    entityToggleGroup,
    rowCount,
    vulnerabilityState,
    pagination,
    sort,
    isUnifiedDeferralsEnabled,
}: CVEsTableContainerProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage } = pagination;
    const { sortOption, getSortParams } = sort;

    const { error, loading, data, previousData } = useQuery(cveListQuery, {
        variables: {
            query: getVulnStateScopedQueryString(querySearchFilter, vulnerabilityState),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
            statusesForExceptionCount: getStatusesForExceptionCount(vulnerabilityState),
        },
    });

    const { data: imageCountData } = useQuery(unfilteredImageCountQuery);

    const { invalidateAll: refetchAll } = useInvalidateVulnerabilityQueries();

    const selectedCves = useMap<string, ExceptionRequestModalProps['cves'][number]>();
    const {
        exceptionRequestModalOptions,
        completedException,
        showModal,
        closeModals,
        createExceptionModalActions,
    } = useExceptionRequestModal();
    const showDeferralUI = isUnifiedDeferralsEnabled && vulnerabilityState === 'OBSERVED';
    const canSelectRows = showDeferralUI;

    const createTableActions = showDeferralUI ? createExceptionModalActions : undefined;

    const tableData = data ?? previousData;
    return (
        <>
            {exceptionRequestModalOptions && (
                <ExceptionRequestModal
                    cves={exceptionRequestModalOptions.cves}
                    type={exceptionRequestModalOptions.type}
                    scopeContext="GLOBAL"
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
            <TableEntityToolbar
                filterToolbar={filterToolbar}
                entityToggleGroup={entityToggleGroup}
                pagination={pagination}
                tableRowCount={rowCount}
                isFiltered={isFiltered}
            >
                {canSelectRows && (
                    <ToolbarItem align={{ default: 'alignRight' }}>
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
                    </ToolbarItem>
                )}
                <ToolbarItem align={{ default: 'alignRight' }} variant="separator" />
            </TableEntityToolbar>
            <Divider component="div" />
            {loading && !tableData && (
                <Bullseye>
                    <Spinner />
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
                        filteredSeverities={searchFilter.SEVERITY as VulnerabilitySeverityLabel[]}
                        selectedCves={selectedCves}
                        canSelectRows={canSelectRows}
                        vulnerabilityState={vulnerabilityState}
                        createTableActions={createTableActions}
                    />
                </div>
            )}
        </>
    );
}

export default CVEsTableContainer;
