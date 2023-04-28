import React from 'react';
import { Flex, Button, ButtonVariant, pluralize, Truncate } from '@patternfly/react-core';
import {
    TableComposable,
    Thead,
    Tr,
    Th,
    Tbody,
    Td,
    ExpandableRowContent,
} from '@patternfly/react-table';

import LinkShim from 'Components/PatternFly/LinkShim';
import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import { graphql } from 'generated/graphql-codegen';
import { DeploymentsForCveFragment } from 'generated/graphql-codegen/graphql';
import { getEntityPagePath } from '../searchUtils';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import EmptyTableResults from '../components/EmptyTableResults';
import DeploymentComponentVulnerabilitiesTable from './DeploymentComponentVulnerabilitiesTable';
import SeverityCountLabels from '../components/SeverityCountLabels';
import DateDistanceTd from '../components/DatePhraseTd';
import { VulnerabilitySeverityLabel } from '../types';

export const deploymentsForCveFragment = graphql(/* GraphQL */ `
    fragment DeploymentsForCVE on Deployment {
        id
        name
        namespace
        clusterName
        created
        lowImageCount: imageCount(query: $lowImageCountQuery)
        moderateImageCount: imageCount(query: $moderateImageCountQuery)
        importantImageCount: imageCount(query: $importantImageCountQuery)
        criticalImageCount: imageCount(query: $criticalImageCountQuery)
        images(query: $query) {
            ...ImageMetadataContext
            imageComponents(query: $query) {
                ...DeploymentComponentVulnerabilities
            }
        }
    }
`);

export type AffectedDeploymentsTableProps = {
    deployments: DeploymentsForCveFragment[];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    filteredSeverities?: VulnerabilitySeverityLabel[];
};

function AffectedDeploymentsTable({
    deployments,
    getSortParams,
    isFiltered,
    filteredSeverities,
}: AffectedDeploymentsTableProps) {
    const expandedRowSet = useSet<string>();
    return (
        // TODO UX question - Collapse to cards, or allow headers to overflow?
        // <TableComposable gridBreakPoint="grid-xl">
        <TableComposable variant="compact">
            <Thead noWrap>
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
            {deployments.length === 0 && <EmptyTableResults colSpan={7} />}
            {deployments.map((deployment, rowIndex) => {
                const {
                    id,
                    name,
                    namespace,
                    clusterName,
                    lowImageCount,
                    moderateImageCount,
                    importantImageCount,
                    criticalImageCount,
                    created,
                    images,
                } = deployment;
                const isExpanded = expandedRowSet.has(id);

                const imageComponentVulns = images.map((image) => ({
                    imageMetadataContext: image,
                    componentVulnerabilities: image.imageComponents,
                }));

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
                                        <Truncate position="middle" content={name} />
                                    </Button>{' '}
                                </Flex>
                            </Td>
                            <Td modifier="nowrap" dataLabel="Images by severity">
                                <SeverityCountLabels
                                    criticalCount={criticalImageCount}
                                    importantCount={importantImageCount}
                                    moderateCount={moderateImageCount}
                                    lowCount={lowImageCount}
                                    filteredSeverities={filteredSeverities}
                                />
                            </Td>
                            <Td dataLabel="Cluster">{clusterName}</Td>
                            <Td dataLabel="Namespace">{namespace}</Td>
                            <Td modifier="nowrap" dataLabel="Images">
                                {pluralize(images.length, 'image')}
                            </Td>

                            <Td modifier="nowrap" dataLabel="First discovered">
                                <DateDistanceTd date={created} />
                            </Td>
                        </Tr>
                        <Tr isExpanded={isExpanded}>
                            <Td />
                            <Td colSpan={6}>
                                <ExpandableRowContent>
                                    <DeploymentComponentVulnerabilitiesTable
                                        images={imageComponentVulns}
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
