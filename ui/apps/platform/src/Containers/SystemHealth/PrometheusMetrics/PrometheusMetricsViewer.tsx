import { useMemo, useState } from 'react';
import type { ReactElement, Ref } from 'react';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Divider,
    Flex,
    FlexItem,
    MenuToggle,
    SearchInput,
    Select,
    SelectList,
    SelectOption,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import { PlusCircleIcon, SyncIcon } from '@patternfly/react-icons';

import useRestQuery from 'hooks/useRestQuery';
import { fetchPrometheusMetrics } from 'services/PrometheusMetricsService';
import { parsePrometheusMetrics } from './prometheusParser';
import MetricTable from './MetricTable';
import { ErrorIcon, SpinnerIcon, SuccessIcon } from '../CardHeaderIcons';

function PrometheusMetricsViewer(): ReactElement {
    const [metricSelectors, setMetricSelectors] = useState<string[]>([]);
    const [isSelectOpen, setIsSelectOpen] = useState(false);
    const [selectedMetricName, setSelectedMetricName] = useState<string>('');
    const [filterValue, setFilterValue] = useState('');

    const { data: metricsText, isLoading, error, refetch } = useRestQuery(fetchPrometheusMetrics);

    const parsedMetrics = useMemo(() => {
        if (!metricsText) {
            return { metrics: {}, metricInfoMap: {}, parseErrors: [] };
        }
        return parsePrometheusMetrics(metricsText);
    }, [metricsText]);

    const metricNames = useMemo(() => {
        return Object.keys(parsedMetrics.metrics).sort();
    }, [parsedMetrics.metrics]);

    const filteredMetricNames = useMemo(() => {
        if (!filterValue) {
            return metricNames;
        }
        return metricNames.filter((name) =>
            name.toLowerCase().includes(filterValue.toLowerCase())
        );
    }, [metricNames, filterValue]);

    const handleAddMetric = () => {
        if (selectedMetricName && !metricSelectors.includes(selectedMetricName)) {
            setMetricSelectors([...metricSelectors, selectedMetricName]);
            setSelectedMetricName('');
            setFilterValue('');
        }
    };

    const handleDeleteMetric = (metricName: string) => {
        setMetricSelectors(metricSelectors.filter((name) => name !== metricName));
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
        return parsedMetrics.metrics[metricName] || [];
    };

    let icon = SpinnerIcon;
    if (isLoading) {
        icon = SpinnerIcon;
    } else if (error) {
        icon = ErrorIcon;
    } else if (metricNames.length > 0) {
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
                            {metricNames.length} metrics available
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
                {!isLoading && !error && parsedMetrics.parseErrors.length > 0 && (
                    <div className="pf-v5-u-warning-color-100 pf-v5-u-mb-md">
                        <strong>Parse warnings:</strong> Failed to parse {parsedMetrics.parseErrors.length} line(s)
                        <ul className="pf-v5-u-mt-sm">
                            {parsedMetrics.parseErrors.slice(0, 5).map((parseError) => (
                                <li key={parseError.lineNumber}>
                                    Line {parseError.lineNumber}: {parseError.line}
                                </li>
                            ))}
                            {parsedMetrics.parseErrors.length > 5 && (
                                <li>... and {parsedMetrics.parseErrors.length - 5} more</li>
                            )}
                        </ul>
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
                                        {filteredMetricNames.map((metricName) => {
                                            const helpText = parsedMetrics.metricInfoMap[metricName];
                                            return (
                                                <SelectOption
                                                    key={metricName}
                                                    value={metricName}
                                                    description={helpText}
                                                >
                                                    {metricName}
                                                </SelectOption>
                                            );
                                        })}
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
                                Select a metric from the dropdown above and click Add to view its
                                data
                            </div>
                        )}

                        {metricSelectors.map((metricName) => {
                            const metricHelp = parsedMetrics.metricInfoMap[metricName];
                            return (
                                <div key={metricName} className="pf-v5-u-mb-md">
                                    <MetricTable
                                        metricName={metricName}
                                        metricHelp={metricHelp}
                                        samples={getSamplesForMetric(metricName)}
                                        onDelete={() => handleDeleteMetric(metricName)}
                                    />
                                </div>
                            );
                        })}
                    </>
                )}
            </CardBody>
        </Card>
    );
}

export default PrometheusMetricsViewer;
