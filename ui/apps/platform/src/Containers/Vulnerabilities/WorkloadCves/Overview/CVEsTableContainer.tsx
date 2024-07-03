import React from 'react';
import { useQuery } from '@apollo/client';
import { Divider, ToolbarItem } from '@patternfly/react-core';
import { DropdownItem } from '@patternfly/react-core/deprecated';

import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useMap from 'hooks/useMap';
import { VulnerabilityState } from 'types/cve.proto';

import { getTableUIState } from 'utils/getTableUIState';
import useHasRequestExceptionsAbility from 'Containers/Vulnerabilities/hooks/useHasRequestExceptionsAbility';
import useInvalidateVulnerabilityQueries from '../../hooks/useInvalidateVulnerabilityQueries';
import CVEsTable, { ImageCVE, cveListQuery, unfilteredImageCountQuery } from '../Tables/CVEsTable';
import { VulnerabilitySeverityLabel } from '../../types';
import { getStatusesForExceptionCount } from '../../utils/searchUtils';
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
    workloadCvesScopedQueryString: string;
    isFiltered: boolean;
};

function CVEsTableContainer({
    filterToolbar,
    entityToggleGroup,
    rowCount,
    vulnerabilityState,
    pagination,
    sort,
    workloadCvesScopedQueryString,
    isFiltered,
}: CVEsTableContainerProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { page, perPage } = pagination;
    const { sortOption, getSortParams } = sort;

    const { error, loading, data } = useQuery<{
        imageCVEs: ImageCVE[];
    }>(cveListQuery, {
        variables: {
            query: workloadCvesScopedQueryString,
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

    const hasRequestExceptionsAbility = useHasRequestExceptionsAbility();

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
        data: data?.imageCVEs,
        error,
        searchFilter,
    });

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
                    <>
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
                        <ToolbarItem align={{ default: 'alignRight' }} variant="separator" />
                    </>
                )}
            </TableEntityToolbar>
            <Divider component="div" />
            <div
                className="workload-cves-table-container"
                role="region"
                aria-live="polite"
                aria-busy={loading ? 'true' : 'false'}
            >
                <CVEsTable
                    tableState={tableState}
                    unfilteredImageCount={imageCountData?.imageCount || 0}
                    getSortParams={getSortParams}
                    isFiltered={isFiltered}
                    filteredSeverities={searchFilter.SEVERITY as VulnerabilitySeverityLabel[]}
                    selectedCves={selectedCves}
                    canSelectRows={canSelectRows}
                    vulnerabilityState={vulnerabilityState}
                    createTableActions={createTableActions}
                    onClearFilters={() => {
                        setSearchFilter({});
                        pagination.setPage(1, 'replace');
                    }}
                />
            </div>
        </>
    );
}

export default CVEsTableContainer;
