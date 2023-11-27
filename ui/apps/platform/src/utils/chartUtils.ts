import { History } from 'react-router-dom';
import {
    ChartBarProps,
    ChartLabelProps,
    ChartThemeColor,
    getTheme,
} from '@patternfly/react-charts';
import merge from 'lodash/merge';

import { policySeverityColorMap } from 'constants/severityColors';
import { ValueOf } from './type.utils';

export const solidBlueChartColor = 'var(--pf-global--palette--blue-400)';

export const severityColorScale = Object.values(policySeverityColorMap);

// Clone default PatternFly chart themes
const defaultTheme = getTheme(ChartThemeColor.multi);

export const defaultChartHeight = 260;

export const defaultChartBarWidth = 18;

const pointerStyles = {
    data: { cursor: 'pointer' },
    labels: { cursor: 'pointer' },
};

/** A Victory chart theme based on grey/yellow/orange/red colors to indicate severity */
export const patternflySeverityTheme = {
    ...defaultTheme,
    bar: {
        style: merge(defaultTheme?.bar?.style ?? {}, pointerStyles),
    },
    stack: {
        ...defaultTheme.stack,
        colorScale: [...severityColorScale],
    },
    legend: {
        ...defaultTheme.legend,
        colorScale: [...severityColorScale],
        style: merge(defaultTheme?.legend?.style ?? {}, pointerStyles),
    },
    tooltip: {
        style: {
            ...(defaultTheme.tooltip?.style ?? {}),
            fontWeight: '600',
            textAnchor: 'start',
        },
        flyoutPadding: { top: 8, bottom: 8, left: 12, right: 12 },
    },
};

type ChartEventProp = NonNullable<ChartBarProps['events']>[number];
type ChartEventHandler = ValueOf<ChartEventProp['eventHandlers']>;

/**
 * A helper function to generate a chart onClick event that initiates navigation to another page.
 */
export function navigateOnClickEvent(
    history: History,
    /** A function that generates the link to navigate to when the entity is clicked */
    linkWith: (props: ChartLabelProps) => string,
    /** An array of Victory onClick event handlers that will be called before navigation is initiated */
    defaultOnClicks: ChartEventHandler[] = []
): ChartEventProp {
    const navigateEventHandler = {
        mutation: (props) => {
            const link = linkWith(props);
            history.push(link);
            return null;
        },
    };

    return {
        target: 'data',
        eventHandlers: {
            onClick: () => [...defaultOnClicks, navigateEventHandler],
        },
    };
}
