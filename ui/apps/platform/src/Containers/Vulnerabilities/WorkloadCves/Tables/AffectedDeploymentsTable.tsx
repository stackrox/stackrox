import React from 'react';
import { Flex, Button, ButtonVariant, pluralize } from '@patternfly/react-core';
import {
    TableComposable,
    Thead,
    Tr,
    Th,
    Tbody,
    Td,
    ExpandableRowContent,
} from '@patternfly/react-table';
import { gql } from '@apollo/client';

import LinkShim from 'Components/PatternFly/LinkShim';
import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';
import { DynamicColumnIcon } from '../DynamicIcon';
import { getEntityPagePath } from '../searchUtils';
import DeploymentComponentVulnerabilitiesLoader from '../DeploymentComponentVulnerabilitiesLoader';

export type DeploymentForCve = {
    id: string;
    name: string;
    namespace: string;
    clusterName: string;
    created: Date | null;
    imageCount: number;
};

export const deploymentsForCveFragment = gql`
    fragment DeploymentsForCVE on Deployment {
        id
        name
        namespace
        clusterName
        created
        imageCount(query: $query)
    }
`;

export type AffectedDeploymentsTableProps = {
    cveId: string;
    deployments: DeploymentForCve[];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

function AffectedDeploymentsTable({
    cveId,
    deployments,
    getSortParams,
    isFiltered,
}: AffectedDeploymentsTableProps) {
    const expandedRowSet = useSet<string>();
    return (
        // TODO UX question - Collapse to cards, or allow headers to overflow?
        // <TableComposable gridBreakPoint="grid-xl">
        <TableComposable>
            <Thead>
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    <Th sort={getSortParams('Deployment')}>Deployment</Th>
                    <Th>
                        Images by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams('Cluster')}>Cluster</Th>
                    <Th sort={getSortParams('Namespace')}>Namespace</Th>
                    <Th>
                        Images
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            {deployments.map((deployment, rowIndex) => {
                const { id, name, namespace, clusterName, imageCount, created } = deployment;
                const isExpanded = expandedRowSet.has(id);

                return (
                    <Tbody key={id} isExpanded={isExpanded}>
                        <Tr>
                            <Td
                                expand={{
                                    rowIndex,
                                    isExpanded,
                                    onToggle: () => expandedRowSet.toggle(id),
                                }}
                            />
                            <Td dataLabel="Deployment">
                                <Flex
                                    direction={{ default: 'column' }}
                                    spaceItems={{ default: 'spaceItemsNone' }}
                                >
                                    <Button
                                        variant={ButtonVariant.link}
                                        isInline
                                        component={LinkShim}
                                        href={getEntityPagePath('Deployment', id)}
                                    >
                                        {name}
                                    </Button>{' '}
                                </Flex>
                            </Td>
                            <Td dataLabel="Images by severity">TODO</Td>
                            <Td dataLabel="Cluster">{clusterName}</Td>
                            <Td dataLabel="Namespace">{namespace}</Td>
                            <Td dataLabel="Images">{pluralize(imageCount, 'image')}</Td>
                            <Td dataLabel="First discovered">
                                {/* TODO Is this the correct field? It differs from the field on the CVE page.
                                Should this be deployment->created or cve->firstDiscovered */}
                                {getDistanceStrictAsPhrase(created, new Date())}
                            </Td>
                        </Tr>
                        <Tr isExpanded={isExpanded}>
                            <Td />
                            <Td colSpan={6}>
                                <ExpandableRowContent>
                                    <DeploymentComponentVulnerabilitiesLoader
                                        isActive={isExpanded}
                                        cveId={cveId}
                                        deploymentId={id}
                                    />
                                </ExpandableRowContent>
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </TableComposable>
    );
}

export default AffectedDeploymentsTable;
