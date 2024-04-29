import React from 'react';
import { gql } from '@apollo/client';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import sortBy from 'lodash/sortBy';

import useTableSort from 'hooks/patternfly/useTableSort';
import { ApiSortOption } from 'types/search';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';

function sortTableData(tableData: NodeComponent[], sortOption: ApiSortOption): NodeComponent[] {
    const sortedRows = sortBy(tableData, (row) => {
        switch (sortOption.field) {
            case 'Component':
                return row.name?.toLowerCase();
            case 'Type':
                return row.source?.toLowerCase();
            case 'Operating system':
                return row.operatingSystem?.toLowerCase();
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
        operatingSystem
        version
        fixedIn
    }
`;

export type NodeComponent = {
    name: string;
    source: string;
    operatingSystem: string;
    version: string;
    fixedIn: string;
};

const sortFields = ['Component', 'Type', 'Operating system'];
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
                    <Th sort={getSortParams('Operating system')}>Operating system</Th>
                    <Th>Version</Th>
                    <Th>Fixed in</Th>
                </Tr>
            </Thead>
            <Tbody>
                {sortedData.map(({ name, source, operatingSystem, version, fixedIn }) => (
                    <Tr key={name}>
                        <Td dataLabel="Component">{name}</Td>
                        <Td dataLabel="Type">{source}</Td>
                        <Td dataLabel="Operating system">{operatingSystem}</Td>
                        <Td dataLabel="Version">{version}</Td>
                        <Td dataLabel="Fixed in">
                            {fixedIn || <VulnerabilityFixableIconText isFixable={false} />}
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </Table>
    );
}

export default NodeComponentsTable;
