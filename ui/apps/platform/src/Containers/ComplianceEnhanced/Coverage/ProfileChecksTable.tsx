import React from 'react';
import { generatePath, Link } from 'react-router-dom';
import {
    Divider,
    Progress,
    ProgressMeasureLocation,
    Text,
    TextVariants,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { Table, TableText, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { ComplianceCheckResultStatusCount } from 'services/ComplianceResultsService';
import { getTableUIState } from 'utils/getTableUIState';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';

import { coverageCheckDetailsPath } from './compliance.coverage.routes';
import CoverageTableViewToggleGroup from './components/CoverageTableViewToggle';
import StatusCountIcon from './components/StatusCountIcon';
import {
    calculateCompliancePercentage,
    getCompliancePfClassName,
    getStatusCounts,
} from './compliance.coverage.utils';

export type ProfileChecksTableProps = {
    isLoading: boolean;
    error: Error | undefined;
    profileChecks: ComplianceCheckResultStatusCount[];
    profileName: string;
};

function ProfileChecksTable({
    isLoading,
    error,
    profileChecks,
    profileName,
}: ProfileChecksTableProps) {
    const tableState = getTableUIState({
        isLoading,
        data: profileChecks,
        error,
        searchFilter: {},
    });

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem>
                        <CoverageTableViewToggleGroup activeToggle="checks" />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider />
            <Table>
                <Thead>
                    <Tr>
                        <Th width={50}>Check</Th>
                        <Th modifier="fitContent">Controls</Th>
                        <Th modifier="fitContent">Fail status</Th>
                        <Th modifier="fitContent">Pass status</Th>
                        <Th modifier="fitContent">Other status</Th>
                        <Th width={40}>Compliance</Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={6}
                    errorProps={{
                        title: 'There was an error loading profile checks',
                    }}
                    emptyProps={{
                        message: 'No results found',
                    }}
                    filteredEmptyProps={{
                        title: 'No checks found',
                        message: 'Clear all filters and try again',
                    }}
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((check) => {
                                const { checkName, rationale, checkStats } = check;
                                const { passCount, failCount, otherCount, totalCount } =
                                    getStatusCounts(checkStats);
                                const passPercentage = calculateCompliancePercentage(
                                    passCount,
                                    totalCount
                                );
                                return (
                                    <Tr key={checkName}>
                                        <Td dataLabel="Check">
                                            <TableText wrapModifier="wrap">
                                                <Link
                                                    to={generatePath(coverageCheckDetailsPath, {
                                                        checkName,
                                                        profileName,
                                                    })}
                                                >
                                                    {checkName}
                                                </Link>
                                            </TableText>
                                            <TableText wrapModifier="truncate">
                                                <Text
                                                    component={TextVariants.small}
                                                    className="pf-v5-u-color-200"
                                                >
                                                    {rationale}
                                                </Text>
                                            </TableText>
                                        </Td>
                                        <Td dataLabel="Controls" modifier="fitContent">
                                            placeholder
                                        </Td>
                                        <Td dataLabel="Fail status" modifier="fitContent">
                                            <StatusCountIcon
                                                text="cluster"
                                                status="fail"
                                                count={failCount}
                                            />
                                        </Td>
                                        <Td dataLabel="Pass status" modifier="fitContent">
                                            <StatusCountIcon
                                                text="cluster"
                                                status="pass"
                                                count={passCount}
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
                                                id={`progress-bar-${checkName}`}
                                                value={passPercentage}
                                                measureLocation={ProgressMeasureLocation.outside}
                                                className={getCompliancePfClassName(passPercentage)}
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
                                                        `progress-bar-${checkName}`
                                                    ) as HTMLButtonElement
                                                }
                                            />
                                        </Td>
                                    </Tr>
                                );
                            })}
                        </Tbody>
                    )}
                />
            </Table>
        </>
    );
}

export default ProfileChecksTable;
