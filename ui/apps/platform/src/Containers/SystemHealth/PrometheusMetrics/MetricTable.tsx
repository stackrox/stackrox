import { useEffect, useMemo, useState } from 'react';
import type { ReactElement } from 'react';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    FlexItem,
    Pagination,
    TextInput,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import type { ThProps } from '@patternfly/react-table';
import { TrashIcon } from '@patternfly/react-icons';

import type { MetricSample } from './types';

type SortDirection = 'asc' | 'desc';

type MetricTableProps = {
    metricName: string;
    metricHelp?: string;
    samples: MetricSample[];
    onDelete: () => void;
};

function MetricTable({
    metricName,
    metricHelp,
    samples,
    onDelete,
}: MetricTableProps): ReactElement {
    const [filters, setFilters] = useState<Record<string, string>>({});
    const [sortColumn, setSortColumn] = useState<string | null>(null);
    const [sortDirection, setSortDirection] = useState<SortDirection>('asc');
    const [page, setPage] = useState(1);
    const [perPage, setPerPage] = useState(20);

    // Extract all unique label names from samples
    const labelNames = useMemo(() => {
        const names = new Set<string>();
        samples.forEach((sample) => {
            Object.keys(sample.labels).forEach((label) => names.add(label));
        });
        return Array.from(names).sort();
    }, [samples]);

    // Filter and sort samples
    const filteredAndSortedSamples = useMemo(() => {
        const filtered = samples.filter((sample) => {
            return Object.entries(filters).every(([labelName, filterValue]) => {
                if (!filterValue) {
                    return true;
                }
                const labelValue = sample.labels[labelName] || '';
                return labelValue.toLowerCase().includes(filterValue.toLowerCase());
            });
        });

        if (!sortColumn) {
            return filtered;
        }

        return [...filtered].sort((a, b) => {
            let aValue: string;
            let bValue: string;

            if (sortColumn === 'value') {
                aValue = a.value;
                bValue = b.value;
            } else {
                aValue = a.labels[sortColumn] ?? '';
                bValue = b.labels[sortColumn] ?? '';
            }

            const comparison = aValue.localeCompare(bValue, undefined, { numeric: true });
            return sortDirection === 'asc' ? comparison : -comparison;
        });
    }, [samples, filters, sortColumn, sortDirection]);

    // Paginate samples
    const paginatedSamples = useMemo(() => {
        const startIndex = (page - 1) * perPage;
        const endIndex = startIndex + perPage;
        return filteredAndSortedSamples.slice(startIndex, endIndex);
    }, [filteredAndSortedSamples, page, perPage]);

    const handleFilterChange = (labelName: string, value: string) => {
        setFilters((prev) => ({
            ...prev,
            [labelName]: value,
        }));
    };

    const clearFilters = () => {
        setFilters({});
    };

    const getSortParams = (columnName: string): ThProps['sort'] => ({
        sortBy: {
            index: sortColumn === columnName ? 0 : undefined,
            direction: sortDirection,
        },
        onSort: (_event, _index, direction) => {
            setSortColumn(columnName);
            setSortDirection(direction);
        },
        columnIndex: 0,
    });

    // Reset to page 1 when filters or sorting changes
    useEffect(() => {
        setPage(1);
    }, [filters, sortColumn, sortDirection]);

    return (
        <Card isCompact>
            <CardHeader>
                <Flex className="pf-v5-u-flex-grow-1">
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <CardTitle component="h3">{metricName}</CardTitle>
                        {metricHelp && (
                            <div className="pf-v5-u-color-200 pf-v5-u-font-size-sm pf-v5-u-mt-xs">
                                {metricHelp}
                            </div>
                        )}
                    </FlexItem>
                    <FlexItem>
                        <Button
                            variant="plain"
                            aria-label="Delete metric"
                            onClick={onDelete}
                            icon={<TrashIcon />}
                        >
                            Delete
                        </Button>
                    </FlexItem>
                </Flex>
            </CardHeader>
            <CardBody>
                {labelNames.length > 0 && (
                    <Toolbar>
                        <ToolbarContent>
                            {labelNames.map((labelName) => (
                                <ToolbarItem key={labelName}>
                                    <TextInput
                                        type="text"
                                        aria-label={`Filter ${labelName}`}
                                        placeholder={`Filter ${labelName}`}
                                        value={filters[labelName] || ''}
                                        onChange={(_event, value) =>
                                            handleFilterChange(labelName, value)
                                        }
                                    />
                                </ToolbarItem>
                            ))}
                            <ToolbarItem>
                                <Button variant="link" onClick={clearFilters}>
                                    Clear filters
                                </Button>
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                )}
                <Table aria-label={`${metricName} metrics table`} variant="compact">
                    <Thead>
                        <Tr>
                            {labelNames.map((labelName) => (
                                <Th key={labelName} sort={getSortParams(labelName)}>
                                    {labelName}
                                </Th>
                            ))}
                            <Th sort={getSortParams('value')}>Value</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {filteredAndSortedSamples.length === 0 ? (
                            <Tr>
                                <Td colSpan={labelNames.length + 1}>
                                    No metrics found
                                    {Object.keys(filters).length > 0 && ' matching filters'}
                                </Td>
                            </Tr>
                        ) : (
                            paginatedSamples.map((sample) => {
                                // Create a stable key from labels and value
                                const key = `${JSON.stringify(sample.labels)}-${sample.value}`;
                                return (
                                    <Tr key={key}>
                                        {labelNames.map((labelName) => (
                                            <Td key={labelName}>
                                                {sample.labels[labelName] ?? '-'}
                                            </Td>
                                        ))}
                                        <Td>{sample.value}</Td>
                                    </Tr>
                                );
                            })
                        )}
                    </Tbody>
                </Table>
                {filteredAndSortedSamples.length > 0 && (
                    <Pagination
                        itemCount={filteredAndSortedSamples.length}
                        perPage={perPage}
                        page={page}
                        onSetPage={(_event, newPage) => setPage(newPage)}
                        onPerPageSelect={(_event, newPerPage) => {
                            setPerPage(newPerPage);
                            setPage(1);
                        }}
                        variant="bottom"
                        isCompact
                    />
                )}
            </CardBody>
        </Card>
    );
}

export default MetricTable;
