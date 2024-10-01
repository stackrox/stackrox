import React from 'react';
import { gql } from '@apollo/client';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import sortBy from 'lodash/sortBy';

import useTableSort from 'hooks/patternfly/useTableSort';
import { ApiSortOptionSingle } from 'types/search';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';

function sortTableData(
    tableData: NodeComponent[],
    sortOption: ApiSortOptionSingle
): NodeComponent[] {
    const sortedRows = sortBy(tableData, (row) => {
        switch (sortOption.field) {
            case 'Component':
                return row.name?.toLowerCase();
            case 'Type':
                return row.source?.toLowerCase();
            default:
                return '';
        }
    });
    if (sortOption.reversed) {
        sortedRows.reverse();
    }
    return sortedRows;
}

export const nodeComponentFragment = gql`
    fragment NodeComponentFragment on NodeComponent {
        name
        source
        version
        nodeVulnerabilities(query: $query) {
            severity
            isFixable
            fixedByVersion
        }
    }
`;

export type NodeComponent = {
    name: string;
    source: string;
    version: string;
    nodeVulnerabilities: {
        severity: string;
        isFixable: boolean;
        fixedByVersion: string;
    }[];
};

const sortFields = ['Component', 'Type'];
const defaultSortOption = { field: 'Component', direction: 'asc' } as const;

export type NodeComponentsTableProps = {
    data: NodeComponent[];
};

function NodeComponentsTable({ data }: NodeComponentsTableProps) {
    const { sortOption, getSortParams } = useTableSort({ sortFields, defaultSortOption });
    const sortedData = sortTableData(data, sortOption);

    // Logically should not occur, but prevents rendering an empty table if it does
    if (data.length === 0) {
        return null;
    }

    return (
        <Table style={{ border: '1px solid var(--pf-v5-c-table--BorderColor)' }}>
            <Thead noWrap>
                <Tr>
                    <Th sort={getSortParams('Component')}>Component</Th>
                    <Th sort={getSortParams('Type')}>Type</Th>
                    <Th>Version</Th>
                    <Th>CVE fixed in</Th>
                </Tr>
            </Thead>
            <Tbody>
                {sortedData.map(({ name, source, version, nodeVulnerabilities }) => {
                    const fixedByVersion = nodeVulnerabilities?.[0]?.fixedByVersion;
                    return (
                        <Tr key={name}>
                            <Td dataLabel="Component">{name}</Td>
                            <Td dataLabel="Type">{source}</Td>
                            <Td dataLabel="Version">{version}</Td>
                            <Td dataLabel="CVE fixed in">
                                {fixedByVersion || (
                                    <VulnerabilityFixableIconText isFixable={false} />
                                )}
                            </Td>
                        </Tr>
                    );
                })}
            </Tbody>
        </Table>
    );
}

export default NodeComponentsTable;
