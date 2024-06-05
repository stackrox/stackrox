import React, { useState } from 'react';
import { generatePath, Link } from 'react-router-dom';
import { Button, ButtonVariant, Text, TextVariants, Tooltip } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import IconText from 'Components/PatternFly/IconText/IconText';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { ComplianceCheckResult } from 'services/ComplianceResultsService';
import { TableUIState } from 'utils/getTableUIState';

import { coverageCheckDetailsPath } from './compliance.coverage.routes';
import { getClusterResultsStatusObject } from './compliance.coverage.utils';
import CheckStatusModal from './components/CheckStatusModal';

export type ClusterDetailsTableProps = {
    clusterId: string;
    profileName: string;
    tableState: TableUIState<ComplianceCheckResult>;
};

function ClusterDetailsTable({ clusterId, profileName, tableState }: ClusterDetailsTableProps) {
    const [selectedCheckResult, setSelectedCheckResult] = useState<ComplianceCheckResult | null>(
        null
    );
    return (
        <>
            <Table>
                <Thead>
                    <Tr>
                        <Th>Check</Th>
                        <Td modifier="fitContent" width={10}>
                            Controls
                        </Td>
                        <Th modifier="fitContent" width={10}>
                            Compliance status
                        </Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={3}
                    errorProps={{
                        title: 'There was an error loading results for this cluster',
                    }}
                    emptyProps={{
                        message: 'No results found for this cluster',
                    }}
                    filteredEmptyProps={{
                        title: 'No results found',
                        message: 'Clear all filters and try again',
                    }}
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((checkResult) => {
                                const { checkName, rationale, status } = checkResult;
                                const clusterStatusObject = getClusterResultsStatusObject(status);

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
                                                    className="pf-v5-u-color-200 pf-v5-u-text-truncate"
                                                >
                                                    {rationale}
                                                </Text>
                                            </div>
                                        </Td>
                                        <Td dataLabel="Controls">placeholder</Td>
                                        <Td dataLabel="Compliance status" modifier="fitContent">
                                            <Tooltip content={clusterStatusObject.tooltipText}>
                                                <Button
                                                    isInline
                                                    variant={ButtonVariant.link}
                                                    onClick={() =>
                                                        setSelectedCheckResult(checkResult)
                                                    }
                                                >
                                                    <IconText
                                                        icon={clusterStatusObject.icon}
                                                        text={clusterStatusObject.statusText}
                                                    />
                                                </Button>
                                            </Tooltip>
                                        </Td>
                                    </Tr>
                                );
                            })}
                        </Tbody>
                    )}
                />
            </Table>
            {selectedCheckResult && (
                <CheckStatusModal
                    checkResult={selectedCheckResult}
                    clusterName={clusterId}
                    isOpen
                    handleClose={() => setSelectedCheckResult(null)}
                />
            )}
        </>
    );
}

export default ClusterDetailsTable;
