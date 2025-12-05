import { useState, useMemo } from 'react';
import type { ReactElement } from 'react';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    TextInput,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Button,
    Flex,
    FlexItem,
} from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { TrashIcon } from '@patternfly/react-icons';

import type { MetricSample } from './types';

type MetricTableProps = {
    metricName: string;
    samples: MetricSample[];
    onDelete: () => void;
};

function MetricTable({ metricName, samples, onDelete }: MetricTableProps): ReactElement {
    const [filters, setFilters] = useState<Record<string, string>>({});

    // Extract all unique label names from samples
    const labelNames = useMemo(() => {
        const names = new Set<string>();
        samples.forEach((sample) => {
            Object.keys(sample.labels).forEach((label) => names.add(label));
        });
        return Array.from(names).sort();
    }, [samples]);

    // Filter samples based on label filters
    const filteredSamples = useMemo(() => {
        return samples.filter((sample) => {
            return Object.entries(filters).every(([labelName, filterValue]) => {
                if (!filterValue) {
                    return true;
                }
                const labelValue = sample.labels[labelName] || '';
                return labelValue.toLowerCase().includes(filterValue.toLowerCase());
            });
        });
    }, [samples, filters]);

    const handleFilterChange = (labelName: string, value: string) => {
        setFilters((prev) => ({
            ...prev,
            [labelName]: value,
        }));
    };

    const clearFilters = () => {
        setFilters({});
    };

    return (
        <Card isCompact>
            <CardHeader>
                <Flex className="pf-v5-u-flex-grow-1">
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <CardTitle component="h3">{metricName}</CardTitle>
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
                                <Th key={labelName}>{labelName}</Th>
                            ))}
                            <Th>Value</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {filteredSamples.length === 0 ? (
                            <Tr>
                                <Td colSpan={labelNames.length + 1}>
                                    No metrics found
                                    {Object.keys(filters).length > 0 && ' matching filters'}
                                </Td>
                            </Tr>
                        ) : (
                            filteredSamples.map((sample, idx) => (
                                <Tr key={idx}>
                                    {labelNames.map((labelName) => (
                                        <Td key={labelName}>{sample.labels[labelName] || '-'}</Td>
                                    ))}
                                    <Td>{sample.value}</Td>
                                </Tr>
                            ))
                        )}
                    </Tbody>
                </Table>
                <div className="pf-v5-u-mt-sm pf-v5-u-color-200">
                    Showing {filteredSamples.length} of {samples.length} samples
                </div>
            </CardBody>
        </Card>
    );
}

export default MetricTable;
