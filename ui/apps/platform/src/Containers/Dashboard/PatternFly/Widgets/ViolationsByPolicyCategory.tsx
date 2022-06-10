import React, { useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Chart,
    ChartAxis,
    ChartStack,
    ChartBar,
    ChartTooltip,
    ChartLabelProps,
} from '@patternfly/react-charts';
import { sortBy } from 'lodash';

import { LinkableChartLabel } from 'Components/PatternFly/Charts/LinkableChartLabel';
import { AlertGroup, Severity } from 'services/AlertsService';
import { severityLabels } from 'messages/common';
import {
    navigateOnClickEvent,
    patternflySeverityTheme,
    defaultChartHeight as chartHeight,
    defaultChartBarWidth,
} from 'utils/chartUtils';
import { getQueryString } from 'utils/queryStringUtils';
import { violationsBasePath } from 'routePaths';
import useResizeObserver from 'hooks/useResizeObserver';
import {
    Dropdown,
    DropdownToggle,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Title,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import useURLSearch from 'hooks/useURLSearch';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import useAlertGroups from '../hooks/useAlertGroups';
import WidgetCard from './WidgetCard';

type CountsBySeverity = {
    Low: Record<string, number>;
    Medium: Record<string, number>;
    High: Record<string, number>;
    Critical: Record<string, number>;
};

function pluckSeverityCount(severity: Severity): (group: AlertGroup) => number {
    return ({ counts }) => {
        const severityCount = counts.find((ct) => ct.severity === severity)?.count || '0';
        return -parseInt(severityCount, 10);
    };
}

function sortByVolume(groups: AlertGroup[]) {
    const sum = (a: number, b: number) => a + b;
    return sortBy(groups, ({ counts }) => {
        return -counts.map(({ count }) => parseInt(count, 10)).reduce(sum);
    });
}

function sortBySeverity(groups: AlertGroup[]) {
    return sortBy(groups, [
        pluckSeverityCount('CRITICAL_SEVERITY'),
        pluckSeverityCount('HIGH_SEVERITY'),
        pluckSeverityCount('MEDIUM_SEVERITY'),
        pluckSeverityCount('LOW_SEVERITY'),
    ]);
}

function getCountsBySeverity(groups: AlertGroup[]): CountsBySeverity {
    const result = {
        Low: {},
        Medium: {},
        High: {},
        Critical: {},
    };

    groups.forEach(({ group, counts }) => {
        result.Low[group] = 0;
        result.Medium[group] = 0;
        result.High[group] = 0;
        result.Critical[group] = 0;

        counts.forEach(({ severity, count }) => {
            result[severityLabels[severity]][group] = parseInt(count, 10);
        });
    });

    return result;
}

function linkForViolationsCategory(category: string) {
    const queryString = getQueryString({
        s: { Category: category },
        sortOption: { field: 'Severity', direction: 'desc' },
    });
    return `${violationsBasePath}${queryString}`;
}

type SortTypeOption = 'Severity' | 'Volume';

type ViolationsByPolicyCategoryChartProps = {
    alertGroups: AlertGroup[];
    sortType: SortTypeOption;
};

const labelLinkCallback = ({ text }: ChartLabelProps) => linkForViolationsCategory(String(text));

const height = `${chartHeight}px` as const;

function ViolationsByPolicyCategoryChart({
    alertGroups,
    sortType,
}: ViolationsByPolicyCategoryChartProps) {
    const history = useHistory();
    const [widgetContainer, setWidgetContainer] = useState<HTMLDivElement | null>(null);
    const widgetContainerResizeEntry = useResizeObserver(widgetContainer);

    const sortedAlertGroups =
        sortType === 'Severity' ? sortBySeverity(alertGroups) : sortByVolume(alertGroups);
    // We reverse here, because PF/Victory charts stack the bars from bottom->up
    const topOrderedGroups = sortedAlertGroups.slice(0, 5).reverse();
    const countsBySeverity = getCountsBySeverity(topOrderedGroups);

    const bars = Object.entries(countsBySeverity).map(([severity, counts]) => {
        const data = Object.entries(counts).map(([group, count]) => ({
            name: severity,
            x: group,
            y: count,
            label: `${severity}: ${count}`,
        }));

        return (
            <ChartBar
                barWidth={defaultChartBarWidth}
                key={severity}
                data={data}
                labelComponent={<ChartTooltip constrainToVisibleArea />}
                events={[
                    navigateOnClickEvent(history, (targetProps) => {
                        const category = targetProps?.datum?.xName;
                        return linkForViolationsCategory(category);
                    }),
                ]}
            />
        );
    });

    return (
        <div ref={setWidgetContainer} style={{ height }}>
            <Chart
                ariaDesc="Number of violation by policy category, grouped by severity"
                ariaTitle="Policy Violations by Category"
                animate={{ duration: 300 }}
                domainPadding={{ x: [20, 20] }}
                legendData={[
                    { name: 'Low' },
                    { name: 'Medium' },
                    { name: 'High' },
                    { name: 'Critical' },
                ]}
                legendPosition="bottom"
                height={chartHeight}
                width={widgetContainerResizeEntry?.contentRect.width} // Victory defaults to 450
                padding={{
                    // TODO Auto-adjust padding based on screen size and/or max text length, if possible
                    left: 180, // left padding is dependent on the length of the text on the left axis
                    bottom: 55, // Adjusted to accommodate legend
                }}
                theme={patternflySeverityTheme}
            >
                <ChartAxis
                    tickLabelComponent={<LinkableChartLabel linkWith={labelLinkCallback} />}
                />
                <ChartAxis dependentAxis />
                <ChartStack horizontal>{bars}</ChartStack>
            </Chart>
        </div>
    );
}

type LifecycleOption = 'All' | 'Deploy' | 'Runtime';

const fieldIdPrefix = 'policy-category-violations';

function ViolationsByPolicyCategory() {
    const { isOpen: isOptionsOpen, onToggle: toggleOptionsOpen } = useSelectToggle();
    const { searchFilter } = useURLSearch();
    const [sortType, sortTypeOption] = useState<SortTypeOption>('Severity');
    const [lifecycle, setLifecycle] = useState<LifecycleOption>('All');

    const queryFilter = { ...searchFilter };
    if (lifecycle === 'Deploy') {
        queryFilter['Lifecycle Stage'] = LIFECYCLE_STAGES.DEPLOY;
    } else if (lifecycle === 'Runtime') {
        queryFilter['Lifecycle Stage'] = LIFECYCLE_STAGES.RUNTIME;
    }
    const query = getRequestQueryStringForSearchFilter(queryFilter);
    const { alertGroups, loading, error } = useAlertGroups('CATEGORY', query);

    return (
        <WidgetCard
            isLoading={loading}
            error={error}
            header={
                <Flex direction={{ default: 'row' }} className="pf-u-pb-md">
                    <FlexItem grow={{ default: 'grow' }}>
                        <Title headingLevel="h2">Policy violations by category</Title>
                    </FlexItem>
                    <FlexItem>
                        <Dropdown
                            toggle={
                                <DropdownToggle
                                    id={`${fieldIdPrefix}-options-toggle`}
                                    toggleVariant="secondary"
                                    onToggle={toggleOptionsOpen}
                                >
                                    Options
                                </DropdownToggle>
                            }
                            position="right"
                            isOpen={isOptionsOpen}
                        >
                            <Form className="pf-u-px-md pf-u-py-sm">
                                <FormGroup fieldId={`${fieldIdPrefix}-sort-by`} label="Sort by">
                                    <ToggleGroup aria-label="Sort data by highest severity counts or highest total violations">
                                        <ToggleGroupItem
                                            className="pf-u-font-weight-normal"
                                            text="Severity"
                                            buttonId={`${fieldIdPrefix}-sort-by-severity`}
                                            isSelected={sortType === 'Severity'}
                                            onChange={() => sortTypeOption('Severity')}
                                        />
                                        <ToggleGroupItem
                                            text="Volume"
                                            buttonId={`${fieldIdPrefix}-sort-by-volume`}
                                            isSelected={sortType === 'Volume'}
                                            onChange={() => sortTypeOption('Volume')}
                                        />
                                    </ToggleGroup>
                                </FormGroup>
                                <FormGroup
                                    fieldId={`${fieldIdPrefix}-lifecycle`}
                                    label="Policy Lifecycle"
                                >
                                    <ToggleGroup aria-label="Filter by policy lifecycle">
                                        <ToggleGroupItem
                                            text="All"
                                            buttonId={`${fieldIdPrefix}-lifecycle-all`}
                                            isSelected={lifecycle === 'All'}
                                            onChange={() => setLifecycle('All')}
                                        />
                                        <ToggleGroupItem
                                            text="Deploy"
                                            buttonId={`${fieldIdPrefix}-lifecycle-deploy`}
                                            isSelected={lifecycle === 'Deploy'}
                                            onChange={() => setLifecycle('Deploy')}
                                        />
                                        <ToggleGroupItem
                                            text="Runtime"
                                            buttonId={`${fieldIdPrefix}-lifecycle-runtime`}
                                            isSelected={lifecycle === 'Runtime'}
                                            onChange={() => setLifecycle('Runtime')}
                                        />
                                    </ToggleGroup>
                                </FormGroup>
                            </Form>
                        </Dropdown>
                    </FlexItem>
                </Flex>
            }
        >
            <ViolationsByPolicyCategoryChart alertGroups={alertGroups} sortType={sortType} />
        </WidgetCard>
    );
}

export default ViolationsByPolicyCategory;
