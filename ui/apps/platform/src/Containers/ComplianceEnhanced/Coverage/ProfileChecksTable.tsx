import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
    Divider,
    Pagination,
    Progress,
    ProgressMeasureLocation,
    Text,
    TextVariants,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseURLSortResult } from 'hooks/useURLSort';
import { ComplianceCheckResultStatusCount } from 'services/ComplianceCommon';
import { TableUIState } from 'utils/getTableUIState';

import { CHECK_NAME_QUERY } from './compliance.coverage.constants';
import { coverageCheckDetailsPath } from './compliance.coverage.routes';
import { calculateCompliancePercentage, getStatusCounts } from './compliance.coverage.utils';
import ControlLabels from './components/ControlLabels';
import ProfilesTableToggleGroup from './components/ProfilesTableToggleGroup';
import StatusCountIcon from './components/StatusCountIcon';
import useScanConfigRouter from './hooks/useScanConfigRouter';

export type ProfileChecksTableProps = {
    profileChecksResultsCount: number;
    profileName: string;
    pagination: UseURLPaginationResult;
    tableState: TableUIState<ComplianceCheckResultStatusCount>;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
};

function ProfileChecksTable({
    profileChecksResultsCount,
    profileName,
    pagination,
    tableState,
    getSortParams,
    onClearFilters,
}: ProfileChecksTableProps) {
    /* eslint-disable no-nested-ternary */
    const { generatePathWithScanConfig } = useScanConfigRouter();
    const [expandedRows, setExpandedRows] = useState<number[]>([]);
    const { page, perPage, setPage, setPerPage } = pagination;

    function toggleRow(selectedRowIndex: number) {
        const newExpandedRows = expandedRows.includes(selectedRowIndex)
            ? expandedRows.filter((index) => index !== selectedRowIndex)
            : [...expandedRows, selectedRowIndex];
        setExpandedRows(newExpandedRows);
    }

    useEffect(() => {
        setExpandedRows([]);
    }, [page, perPage, tableState]);

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem>
                        <ProfilesTableToggleGroup activeToggle="checks" />
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                        <Pagination
                            itemCount={profileChecksResultsCount}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider />
            <Table>
                <Thead>
                    <Tr>
                        <Th sort={getSortParams(CHECK_NAME_QUERY)} width={60}>
                            Check
                        </Th>
                        <Th modifier="fitContent">Controls</Th>
                        <Th modifier="fitContent">Pass status</Th>
                        <Th modifier="fitContent">Fail status</Th>
                        <Th modifier="fitContent">Manual status</Th>
                        <Th modifier="fitContent">Other status</Th>
                        <Th width={40}>Compliance</Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={7}
                    errorProps={{
                        title: 'There was an error loading profile checks',
                    }}
                    emptyProps={{
                        message:
                            'If you have recently created a scan schedule, please wait a few minutes for the results to become available.',
                    }}
                    filteredEmptyProps={{ onClearFilters }}
                    renderer={({ data }) => (
                        <>
                            {data.map((check, rowIndex) => {
                                const { checkName, rationale, checkStats, controls } = check;
                                const {
                                    passCount,
                                    failCount,
                                    manualCount,
                                    otherCount,
                                    totalCount,
                                } = getStatusCounts(checkStats);
                                const passPercentage = calculateCompliancePercentage(
                                    passCount,
                                    totalCount
                                );
                                const progressBarId = `progress-bar-${checkName}`;
                                const isRowExpanded = expandedRows.includes(rowIndex);

                                return (
                                    <Tbody isExpanded={isRowExpanded} key={checkName}>
                                        <Tr>
                                            <Td dataLabel="Check">
                                                <Link
                                                    to={generatePathWithScanConfig(
                                                        coverageCheckDetailsPath,
                                                        {
                                                            checkName,
                                                            profileName,
                                                        }
                                                    )}
                                                >
                                                    {checkName}
                                                </Link>
                                                {/*
                                                grid display is required to prevent the cell from
                                                expanding to the text length. The Truncate PF component
                                                is not used here because it displays a tooltip on hover
                                            */}
                                                <div style={{ display: 'grid' }}>
                                                    <Text
                                                        component={TextVariants.small}
                                                        className="pf-v5-u-color-200 pf-v5-u-text-truncate"
                                                    >
                                                        {rationale}
                                                    </Text>
                                                </div>
                                            </Td>
                                            <Td
                                                dataLabel="Controls"
                                                modifier="fitContent"
                                                compoundExpand={
                                                    controls.length > 1
                                                        ? {
                                                              isExpanded: isRowExpanded,
                                                              onToggle: () => toggleRow(rowIndex),
                                                              rowIndex,
                                                              columnIndex: 1,
                                                          }
                                                        : undefined
                                                }
                                            >
                                                {controls.length > 1 ? (
                                                    `${controls.length} controls`
                                                ) : controls.length === 1 ? (
                                                    <ControlLabels controls={controls} />
                                                ) : (
                                                    '-'
                                                )}
                                            </Td>
                                            <Td dataLabel="Pass status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="cluster"
                                                    status="pass"
                                                    count={passCount}
                                                />
                                            </Td>
                                            <Td dataLabel="Fail status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="cluster"
                                                    status="fail"
                                                    count={failCount}
                                                />
                                            </Td>
                                            <Td dataLabel="Manual status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="cluster"
                                                    status="manual"
                                                    count={manualCount}
                                                />
                                            </Td>
                                            <Td dataLabel="Other status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="cluster"
                                                    status="other"
                                                    count={otherCount}
                                                />
                                            </Td>
                                            <Td dataLabel="Compliance">
                                                <Progress
                                                    id={progressBarId}
                                                    value={passPercentage}
                                                    measureLocation={
                                                        ProgressMeasureLocation.outside
                                                    }
                                                    aria-label={`${checkName} compliance percentage`}
                                                />
                                                <Tooltip
                                                    content={
                                                        <div>
                                                            {`${passCount} / ${totalCount} clusters are passing this check`}
                                                        </div>
                                                    }
                                                    triggerRef={() =>
                                                        document.getElementById(
                                                            progressBarId
                                                        ) as HTMLButtonElement
                                                    }
                                                />
                                            </Td>
                                        </Tr>
                                        {isRowExpanded && (
                                            <Tr isExpanded={isRowExpanded}>
                                                <Td colSpan={7}>
                                                    <ExpandableRowContent>
                                                        <ControlLabels controls={controls} />
                                                    </ExpandableRowContent>
                                                </Td>
                                            </Tr>
                                        )}
                                    </Tbody>
                                );
                            })}
                        </>
                    )}
                />
            </Table>
        </>
    );
}

export default ProfileChecksTable;
