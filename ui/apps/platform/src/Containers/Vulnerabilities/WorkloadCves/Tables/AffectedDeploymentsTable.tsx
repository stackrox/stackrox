import React from 'react';
import { Link } from 'react-router-dom';
import { Flex, pluralize, Truncate } from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td, ExpandableRowContent } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import { VulnerabilityState } from 'types/cve.proto';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { TableUIState } from 'utils/getTableUIState';
import ExpandRowTh from 'Components/ExpandRowTh';
import {
    generateVisibilityForColumns,
    getHiddenColumnCount,
    ManagedColumns,
} from 'hooks/useManagedColumns';
import { getWorkloadEntityPagePath } from '../../utils/searchUtils';
import DeploymentComponentVulnerabilitiesTable, {
    DeploymentComponentVulnerability,
    ImageMetadataContext,
    deploymentComponentVulnerabilitiesFragment,
    imageMetadataContextFragment,
} from './DeploymentComponentVulnerabilitiesTable';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { VulnerabilitySeverityLabel } from '../../types';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

export const tableId = 'WorkloadCvesAffectedDeploymentsTable';
export const defaultColumns = {
    imagesBySeverity: {
        title: 'Images by severity',
        isShownByDefault: true,
    },
    cluster: {
        title: 'Cluster',
        isShownByDefault: true,
    },
    namespace: {
        title: 'Namespace',
        isShownByDefault: true,
    },
    images: {
        title: 'Images',
        isShownByDefault: true,
    },
    firstDiscovered: {
        title: 'First discovered',
        isShownByDefault: true,
    },
} as const;

export type DeploymentForCve = {
    id: string;
    name: string;
    namespace: string;
    clusterName: string;
    created: string | null;
    unknownImageCount: number;
    lowImageCount: number;
    moderateImageCount: number;
    importantImageCount: number;
    criticalImageCount: number;
    images: (ImageMetadataContext & { imageComponents: DeploymentComponentVulnerability[] })[];
};

export const deploymentsForCveFragment = gql`
    ${imageMetadataContextFragment}
    ${deploymentComponentVulnerabilitiesFragment}
    fragment DeploymentsForCVE on Deployment {
        id
        name
        namespace
        clusterName
        created
        unknownImageCount: imageCount(query: $unknownImageCountQuery)
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
`;

export type AffectedDeploymentsTableProps = {
    tableState: TableUIState<DeploymentForCve>;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    cve: string;
    vulnerabilityState: VulnerabilityState;
    filteredSeverities?: VulnerabilitySeverityLabel[];
    onClearFilters: () => void;
    tableConfig: ManagedColumns<keyof typeof defaultColumns>['columns'];
};

function AffectedDeploymentsTable({
    tableState,
    getSortParams,
    isFiltered,
    cve,
    vulnerabilityState,
    filteredSeverities,
    onClearFilters,
    tableConfig,
}: AffectedDeploymentsTableProps) {
    const { getAbsoluteUrl } = useWorkloadCveViewContext();
    const getVisibilityClass = generateVisibilityForColumns(tableConfig);
    const hiddenColumnCount = getHiddenColumnCount(tableConfig);
    const expandedRowSet = useSet<string>();

    const colSpan = 8 + -hiddenColumnCount;
    return (
        <Table variant="compact">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh />
                    <Th sort={getSortParams('Deployment')}>Deployment</Th>
                    <Th className={getVisibilityClass('imagesBySeverity')}>
                        Images by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th className={getVisibilityClass('cluster')} sort={getSortParams('Cluster')}>
                        Cluster
                    </Th>
                    <Th
                        className={getVisibilityClass('namespace')}
                        sort={getSortParams('Namespace')}
                    >
                        Namespace
                    </Th>
                    <Th className={getVisibilityClass('images')}>
                        Images
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th className={getVisibilityClass('firstDiscovered')}>First discovered</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{ message: 'No deployments were found that are affected by this CVE' }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((deployment, rowIndex) => {
                        const {
                            id,
                            name,
                            namespace,
                            clusterName,
                            unknownImageCount,
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
                                            <Link
                                                to={getAbsoluteUrl(
                                                    getWorkloadEntityPagePath(
                                                        'Deployment',
                                                        id,
                                                        vulnerabilityState
                                                    )
                                                )}
                                            >
                                                <Truncate position="middle" content={name} />
                                            </Link>
                                        </Flex>
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('imagesBySeverity')}
                                        modifier="nowrap"
                                        dataLabel="Images by severity"
                                    >
                                        <SeverityCountLabels
                                            criticalCount={criticalImageCount}
                                            importantCount={importantImageCount}
                                            moderateCount={moderateImageCount}
                                            lowCount={lowImageCount}
                                            unknownCount={unknownImageCount}
                                            filteredSeverities={filteredSeverities}
                                        />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cluster')}
                                        dataLabel="Cluster"
                                    >
                                        {clusterName}
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('namespace')}
                                        dataLabel="Namespace"
                                    >
                                        {namespace}
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('images')}
                                        modifier="nowrap"
                                        dataLabel="Images"
                                    >
                                        {pluralize(images.length, 'image')}
                                    </Td>

                                    <Td
                                        className={getVisibilityClass('firstDiscovered')}
                                        modifier="nowrap"
                                        dataLabel="First discovered"
                                    >
                                        <DateDistance date={created} />
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={6}>
                                        <ExpandableRowContent>
                                            <DeploymentComponentVulnerabilitiesTable
                                                images={imageComponentVulns}
                                                cve={cve}
                                                vulnerabilityState={vulnerabilityState}
                                            />
                                        </ExpandableRowContent>
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })
                }
            />
        </Table>
    );
}

export default AffectedDeploymentsTable;
