import React, { useState, useCallback } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Chart,
    ChartAxis,
    ChartStack,
    ChartBar,
    ChartTooltip,
    ChartLabelProps,
    ChartLegend,
    getInteractiveLegendEvents,
    getInteractiveLegendItemStyles,
} from '@patternfly/react-charts';
import sortBy from 'lodash/sortBy';

import { LinkableChartLabel } from 'Components/PatternFly/Charts/LinkableChartLabel';
import { AlertGroup } from 'services/AlertsService';
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
    LifecycleStage,
    policySeverities as severitiesLowToCritical,
    PolicySeverity,
} from 'types/policy.proto';

import { SearchFilter } from 'types/search';

/**
 * This function iterates an array of AlertGroups and zeros out severities that
 * have been filtered by the user in the widget's legend.
 */
function zeroOutFilteredSeverities(
    groups: AlertGroup[],
    hiddenSeverities: Set<PolicySeverity>
): AlertGroup[] {
    return groups.map(({ group, counts }) => ({
        group,
        counts: counts.map(({ severity, count }) => ({
            severity,
            count: hiddenSeverities.has(severity) ? '0' : count,
        })),
    }));
}

function pluckSeverityCount(severity: PolicySeverity): (group: AlertGroup) => number {
    return ({ counts }) => {
        const severityCount = counts.find((ct) => ct.severity === severity)?.count ?? '0';
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

type CountsBySeverity = Record<PolicySeverity, Record<string, number>>;

function getCountsBySeverity(groups: AlertGroup[]): CountsBySeverity {
    const result = {
        LOW_SEVERITY: {},
        MEDIUM_SEVERITY: {},
        HIGH_SEVERITY: {},
        CRITICAL_SEVERITY: {},
    };

    groups.forEach(({ group, counts }) => {
        result.LOW_SEVERITY[group] = 0;
        result.MEDIUM_SEVERITY[group] = 0;
        result.HIGH_SEVERITY[group] = 0;
        result.CRITICAL_SEVERITY[group] = 0;

        counts.forEach(({ severity, count }) => {
            result[severity][group] = parseInt(count, 10);
        });
    });

    return result;
}

function linkForViolationsCategory(
    category: string,
    searchFilter: SearchFilter,
    lifecycle: LifecycleOption,
    hiddenSeverities: Set<PolicySeverity>
) {
    const search: SearchFilter = {
        ...searchFilter,
        Category: category,
    };

    if (lifecycle !== 'ALL') {
        search['Lifecycle Stage'] = lifecycle;
    }
    if (hiddenSeverities.size > 0) {
        search.Severity = severitiesLowToCritical.filter((s) => !hiddenSeverities.has(s));
    }

    const queryString = getQueryString({
        s: search,
        sortOption: { field: 'Severity', direction: 'desc' },
    });
    return `${violationsBasePath}${queryString}`;
}

function tooltipForCategory(
    category: string,
    countsBySeverity: CountsBySeverity,
    hiddenSeverities: Set<PolicySeverity>
): string {
    return severitiesLowToCritical
        .filter((severity) => !hiddenSeverities.has(severity))
        .map((severity) => `${severityLabels[severity]}: ${countsBySeverity[severity][category]}`)
        .join('\n');
}

const chartTheme = patternflySeverityTheme;

type SortTypeOption = 'Severity' | 'Total';

type LifecycleOption = 'ALL' | Exclude<LifecycleStage, 'BUILD'>;

export type Config = {
    sortType: SortTypeOption;
    lifecycle: LifecycleOption;
    hiddenSeverities: Readonly<PolicySeverity[]>;
};

type ViolationsByPolicyCategoryChartProps = {
    alertGroups: AlertGroup[];
    sortType: SortTypeOption;
    lifecycle: LifecycleOption;
    searchFilter: SearchFilter;
    hiddenSeverities: Set<PolicySeverity>;
    setHiddenSeverities: (severities: Set<PolicySeverity>) => Promise<Config>;
};

function ViolationsByPolicyCategoryChart({
    alertGroups,
    sortType,
    lifecycle,
    hiddenSeverities,
    setHiddenSeverities,
    searchFilter,
}: ViolationsByPolicyCategoryChartProps) {
    const history = useHistory();
    const [widgetContainer, setWidgetContainer] = useState<HTMLDivElement | null>(null);
    const widgetContainerResizeEntry = useResizeObserver(widgetContainer);

    const labelLinkCallback = useCallback(
        ({ text }: ChartLabelProps) =>
            linkForViolationsCategory(String(text), searchFilter, lifecycle, hiddenSeverities),
        [searchFilter, lifecycle, hiddenSeverities]
    );

    const filteredAlertGroups = zeroOutFilteredSeverities(alertGroups, hiddenSeverities);
    const sortedAlertGroups =
        sortType === 'Severity'
            ? sortBySeverity(filteredAlertGroups)
            : sortByVolume(filteredAlertGroups);
    // We reverse here, because PF/Victory charts stack the bars from bottom->up
    const topOrderedGroups = sortedAlertGroups.slice(0, 5).reverse();
    const countsBySeverity = getCountsBySeverity(topOrderedGroups);

    const bars = severitiesLowToCritical.map((severity) => {
        const counts = countsBySeverity[severity];
        const data = Object.entries(counts).map(([group, count]) => ({
            name: severity,
            x: group,
            y: count,
            label: tooltipForCategory(group, countsBySeverity, hiddenSeverities),
        }));

        return (
            <ChartBar
                barWidth={defaultChartBarWidth}
                key={severity}
                data={data}
                labelComponent={<ChartTooltip constrainToVisibleArea />}
                events={[
                    // TS2339: Property 'xName' does not exist on type '{}'.
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    navigateOnClickEvent(history, (targetProps: any) => {
                        const category = targetProps?.datum?.xName;
                        return linkForViolationsCategory(
                            category,
                            searchFilter,
                            lifecycle,
                            hiddenSeverities
                        );
                    }),
                ]}
            />
        );
    });

    function getLegendData() {
        const legendData = severitiesLowToCritical.map((severity) => {
            return {
                name: severityLabels[severity],
                ...getInteractiveLegendItemStyles(hiddenSeverities.has(severity)),
            };
        });
        return legendData;
    }

    function onLegendClick({ index }: { index: number }) {
        const newHidden = new Set(hiddenSeverities);
        const targetSeverity = severitiesLowToCritical[index];
        if (newHidden.has(targetSeverity)) {
            newHidden.delete(targetSeverity);
            // Do not allow the user to disable all severities
        } else if (hiddenSeverities.size < 3) {
            newHidden.add(targetSeverity);
        }
        return setHiddenSeverities(newHidden);
    }

    return (
        <div ref={setWidgetContainer}>
            <Chart
                ariaDesc="Number of violations by policy category, grouped by severity"
                animate={{ duration: 300 }}
                domainPadding={{ x: [20, 20] }}
                events={getInteractiveLegendEvents({
                    chartNames: [Object.values(severityLabels)],
                    isHidden: (index) => hiddenSeverities.has(severitiesLowToCritical[index]),
                    legendName: 'legend',
                    onLegendClick,
                })}
                legendComponent={<ChartLegend name="legend" data={getLegendData()} />}
                legendPosition="bottom"
                height={chartHeight}
                width={widgetContainerResizeEntry?.contentRect.width} // Victory defaults to 450
                padding={{
                    // TODO Auto-adjust padding based on screen size and/or max text length, if possible
                    left: 180, // left padding is dependent on the length of the text on the left axis
                    bottom: 55, // Adjusted to accommodate legend
                    right: 35,
                }}
                theme={chartTheme}
            >
                <ChartAxis
                    tickLabelComponent={<LinkableChartLabel linkWith={labelLinkCallback} />}
                />
                <ChartAxis dependentAxis fixLabelOverlap tickFormat={String} />
                <ChartStack horizontal>{bars}</ChartStack>
            </Chart>
        </div>
    );
}

export default ViolationsByPolicyCategoryChart;
