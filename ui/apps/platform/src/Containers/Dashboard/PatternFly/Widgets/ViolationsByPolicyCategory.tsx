import React, { useState, useMemo, useCallback } from 'react';
import { useLocation, useHistory } from 'react-router-dom';
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
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import useURLSearch from 'hooks/useURLSearch';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import {
    LifecycleStage,
    policySeverities as severitiesLowToCritical,
    PolicySeverity,
} from 'types/policy.proto';

import { SearchFilter } from 'types/search';
import useWidgetConfig from 'hooks/useWidgetConfig';
import useAlertGroups from '../hooks/useAlertGroups';
import WidgetCard from './WidgetCard';
import NoDataEmptyState from './NoDataEmptyState';

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

function linkForViolationsCategory(category: string, searchFilter: SearchFilter) {
    const queryString = getQueryString({
        s: { ...searchFilter, Category: category },
        sortOption: { field: 'Severity', direction: 'desc' },
    });
    return `${violationsBasePath}${queryString}`;
}

type SortTypeOption = 'Severity' | 'Total';

type ViolationsByPolicyCategoryChartProps = {
    alertGroups: AlertGroup[];
    sortType: SortTypeOption;
    searchFilter: SearchFilter;
    hiddenSeverities: Set<PolicySeverity>;
    setHiddenSeverities: (severities: Set<PolicySeverity>) => Promise<Config>;
};

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

function ViolationsByPolicyCategoryChart({
    alertGroups,
    sortType,
    hiddenSeverities,
    setHiddenSeverities,
    searchFilter,
}: ViolationsByPolicyCategoryChartProps) {
    const history = useHistory();
    const [widgetContainer, setWidgetContainer] = useState<HTMLDivElement | null>(null);
    const widgetContainerResizeEntry = useResizeObserver(widgetContainer);

    const labelLinkCallback = useCallback(
        ({ text }: ChartLabelProps) => linkForViolationsCategory(String(text), searchFilter),
        [searchFilter]
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
                    navigateOnClickEvent(history, (targetProps) => {
                        const category = targetProps?.datum?.xName;
                        return linkForViolationsCategory(category, searchFilter);
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
                <ChartAxis dependentAxis tickFormat={String} />
                <ChartStack horizontal>{bars}</ChartStack>
            </Chart>
        </div>
    );
}

type LifecycleOption = 'ALL' | Exclude<LifecycleStage, 'BUILD'>;

const fieldIdPrefix = 'policy-category-violations';

type Config = {
    sortType: SortTypeOption;
    lifecycle: LifecycleOption;
    hiddenSeverities: Readonly<PolicySeverity[]>;
};

const defaultHiddenSeverities = ['MEDIUM_SEVERITY', 'LOW_SEVERITY'] as const;

const defaultConfig = {
    sortType: 'Severity',
    lifecycle: 'ALL',
    hiddenSeverities: defaultHiddenSeverities,
} as const;

function ViolationsByPolicyCategory() {
    const { isOpen: isOptionsOpen, onToggle: toggleOptionsOpen } = useSelectToggle();
    const { pathname } = useLocation();
    const { searchFilter } = useURLSearch();

    const [{ sortType, lifecycle, hiddenSeverities }, updateConfig] = useWidgetConfig<Config>(
        'ViolationsByPolicyCategory',
        pathname,
        defaultConfig
    );

    const hiddenSeveritySet = useMemo(() => new Set(hiddenSeverities), [hiddenSeverities]);

    const onHiddenSeverityUpdate = useCallback(
        (newHidden: Set<PolicySeverity>) =>
            updateConfig({ hiddenSeverities: Array.from(newHidden) }),
        [updateConfig]
    );

    const queryFilter = { ...searchFilter };
    if (lifecycle === 'DEPLOY') {
        queryFilter['Lifecycle Stage'] = LIFECYCLE_STAGES.DEPLOY;
    } else if (lifecycle === 'RUNTIME') {
        queryFilter['Lifecycle Stage'] = LIFECYCLE_STAGES.RUNTIME;
    }
    const query = getRequestQueryStringForSearchFilter(queryFilter);
    const { data: alertGroups, loading, error } = useAlertGroups(query, 'CATEGORY');

    return (
        <WidgetCard
            isLoading={loading}
            error={error}
            header={
                <Flex direction={{ default: 'row' }}>
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
                                            onChange={() => updateConfig({ sortType: 'Severity' })}
                                        />
                                        <ToggleGroupItem
                                            text="Total"
                                            buttonId={`${fieldIdPrefix}-sort-by-total`}
                                            isSelected={sortType === 'Total'}
                                            onChange={() => updateConfig({ sortType: 'Total' })}
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
                                            isSelected={lifecycle === 'ALL'}
                                            onChange={() => updateConfig({ lifecycle: 'ALL' })}
                                        />
                                        <ToggleGroupItem
                                            text="Deploy"
                                            buttonId={`${fieldIdPrefix}-lifecycle-deploy`}
                                            isSelected={lifecycle === 'DEPLOY'}
                                            onChange={() => updateConfig({ lifecycle: 'DEPLOY' })}
                                        />
                                        <ToggleGroupItem
                                            text="Runtime"
                                            buttonId={`${fieldIdPrefix}-lifecycle-runtime`}
                                            isSelected={lifecycle === 'RUNTIME'}
                                            onChange={() => updateConfig({ lifecycle: 'RUNTIME' })}
                                        />
                                    </ToggleGroup>
                                </FormGroup>
                            </Form>
                        </Dropdown>
                    </FlexItem>
                </Flex>
            }
        >
            {alertGroups && alertGroups.length > 0 ? (
                <ViolationsByPolicyCategoryChart
                    alertGroups={alertGroups}
                    sortType={sortType}
                    searchFilter={searchFilter}
                    hiddenSeverities={hiddenSeveritySet}
                    setHiddenSeverities={onHiddenSeverityUpdate}
                />
            ) : (
                <NoDataEmptyState />
            )}
        </WidgetCard>
    );
}

export default ViolationsByPolicyCategory;
