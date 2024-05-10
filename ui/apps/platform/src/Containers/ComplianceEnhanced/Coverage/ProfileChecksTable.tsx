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
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { ComplianceCheckResultStatusCount } from 'services/ComplianceResultsService';
import { getTableUIState } from 'utils/getTableUIState';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';

import { coverageCheckDetailsPath } from './compliance.coverage.routes';
import ProfilesTableToggleGroup from './components/ProfilesTableToggleGroup';
import StatusCountIcon from './components/StatusCountIcon';
import {
    calculateCompliancePercentage,
    getCompliancePfClassName,
    getStatusCounts,
} from './compliance.coverage.utils';

import './ProfileChecksTable.css';

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
                        <ProfilesTableToggleGroup activeToggle="checks" />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider />
            <Table>
                <Thead>
                    <Tr>
                        <Th width={60}>Check</Th>
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
                                const progressBarId = `progress-bar-${checkName}`;

                                return (
                                    <Tr key={checkName}>
                                        <Td dataLabel="Check">
                                            <Link
                                                to={generatePath(coverageCheckDetailsPath, {
                                                    checkName,
                                                    profileName,
                                                })}
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
                                                    className="pf-v5-u-color-200 truncate-text"
                                                >
                                                    {rationale}
                                                </Text>
                                            </div>
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
                                                id={progressBarId}
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
                                                        progressBarId
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
