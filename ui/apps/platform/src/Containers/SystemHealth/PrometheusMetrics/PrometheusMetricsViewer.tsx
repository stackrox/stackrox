import { useState, useMemo } from 'react';
import type { ReactElement, Ref } from 'react';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Button,
    Select,
    SelectList,
    SelectOption,
    MenuToggle,
    SearchInput,
    Flex,
    FlexItem,
    Divider,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import { PlusCircleIcon, SyncIcon } from '@patternfly/react-icons';

import useRestQuery from 'hooks/useRestQuery';
import { fetchPrometheusMetrics } from 'services/PrometheusMetricsService';
import { parsePrometheusMetrics } from './prometheusParser';
import MetricTable from './MetricTable';
import type { MetricSelector } from './types';
import { ErrorIcon, SpinnerIcon, SuccessIcon } from '../CardHeaderIcons';

function PrometheusMetricsViewer(): ReactElement {
    const [metricSelectors, setMetricSelectors] = useState<MetricSelector[]>([]);
    const [isSelectOpen, setIsSelectOpen] = useState(false);
    const [selectedMetricName, setSelectedMetricName] = useState<string>('');
    const [filterValue, setFilterValue] = useState('');

    const { data: metricsText, isLoading, error, refetch } = useRestQuery(fetchPrometheusMetrics);

    const parsedMetrics = useMemo(() => {
        if (!metricsText) {
            return { metrics: [], metricNames: [] };
        }
        return parsePrometheusMetrics(metricsText);
    }, [metricsText]);

    const filteredMetricNames = useMemo(() => {
        if (!filterValue) {
            return parsedMetrics.metricNames;
        }
        return parsedMetrics.metricNames.filter((name) =>
            name.toLowerCase().includes(filterValue.toLowerCase())
        );
    }, [parsedMetrics.metricNames, filterValue]);

    const handleAddMetric = () => {
        if (selectedMetricName) {
            const newSelector: MetricSelector = {
                id: `${selectedMetricName}-${Date.now()}`,
                metricName: selectedMetricName,
            };
            setMetricSelectors([...metricSelectors, newSelector]);
            setSelectedMetricName('');
            setFilterValue('');
        }
    };

    const handleDeleteMetric = (id: string) => {
        setMetricSelectors(metricSelectors.filter((selector) => selector.id !== id));
    };

    const handleMetricSelect = (_event, selection: string | number | undefined) => {
        if (typeof selection === 'string') {
            setSelectedMetricName(selection);
            setIsSelectOpen(false);
        }
    };

    const onToggle = () => {
        const willBeOpen = !isSelectOpen;
        setIsSelectOpen(willBeOpen);
        if (!willBeOpen) {
            setFilterValue('');
        }
    };

    const getSamplesForMetric = (metricName: string) => {
        return parsedMetrics.metrics.filter((sample) => sample.metricName === metricName);
    };

    let icon = SpinnerIcon;
    if (isLoading) {
        icon = SpinnerIcon;
    } else if (error) {
        icon = ErrorIcon;
    } else if (parsedMetrics.metricNames.length > 0) {
        icon = SuccessIcon;
    }

    return (
        <Card isCompact>
            <CardHeader>
                <Flex className="pf-v5-u-flex-grow-1">
                    <FlexItem>{icon}</FlexItem>
                    <FlexItem>
                        <CardTitle component="h2">Prometheus Metrics Viewer</CardTitle>
                    </FlexItem>
                    {!isLoading && !error && (
                        <FlexItem align={{ default: 'alignRight' }}>
                            {parsedMetrics.metricNames.length} metrics available
                        </FlexItem>
                    )}
                    <FlexItem align={{ default: 'alignRight' }}>
                        <Button
                            variant="plain"
                            aria-label="Refresh metrics"
                            onClick={refetch}
                            isDisabled={isLoading}
                            icon={<SyncIcon />}
                        >
                            Refresh
                        </Button>
                    </FlexItem>
                </Flex>
            </CardHeader>
            <Divider />
            <CardBody>
                {error && (
                    <div className="pf-v5-u-danger-color-100 pf-v5-u-mb-md">
                        Error loading metrics: {error.message}
                    </div>
                )}
                {!isLoading && !error && (
                    <>
                        <Flex className="pf-v5-u-mb-md">
                            <FlexItem flex={{ default: 'flex_1' }}>
                                <Select
                                    isOpen={isSelectOpen}
                                    selected={selectedMetricName}
                                    onSelect={handleMetricSelect}
                                    onOpenChange={setIsSelectOpen}
                                    toggle={(toggleRef: Ref<MenuToggleElement>) => (
                                        <MenuToggle
                                            ref={toggleRef}
                                            onClick={onToggle}
                                            isExpanded={isSelectOpen}
                                            style={{ width: '100%' }}
                                        >
                                            {selectedMetricName || 'Select a metric to view'}
                                        </MenuToggle>
                                    )}
                                >
                                    <SelectList style={{ maxHeight: '400px', overflow: 'auto' }}>
                                        <div className="pf-v5-u-p-md">
                                            <SearchInput
                                                value={filterValue}
                                                onChange={(_event, value) => setFilterValue(value)}
                                                placeholder="Filter metrics"
                                                aria-label="Filter metrics"
                                            />
                                        </div>
                                        <Divider />
                                        {filteredMetricNames.map((metricName) => (
                                            <SelectOption key={metricName} value={metricName}>
                                                {metricName}
                                            </SelectOption>
                                        ))}
                                    </SelectList>
                                </Select>
                            </FlexItem>
                            <FlexItem>
                                <Button
                                    variant="primary"
                                    icon={<PlusCircleIcon />}
                                    onClick={handleAddMetric}
                                    isDisabled={!selectedMetricName}
                                >
                                    Add
                                </Button>
                            </FlexItem>
                        </Flex>

                        {metricSelectors.length === 0 && (
                            <div className="pf-v5-u-color-200">
                                Select a metric from the dropdown above and click Add to view its data
                            </div>
                        )}

                        {metricSelectors.map((selector) => (
                            <div key={selector.id} className="pf-v5-u-mb-md">
                                <MetricTable
                                    metricName={selector.metricName}
                                    samples={getSamplesForMetric(selector.metricName)}
                                    onDelete={() => handleDeleteMetric(selector.id)}
                                />
                            </div>
                        ))}
                    </>
                )}
            </CardBody>
        </Card>
    );
}

export default PrometheusMetricsViewer;
