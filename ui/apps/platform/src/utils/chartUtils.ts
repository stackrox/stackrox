import { History } from 'react-router-dom';
import { getTheme, ChartThemeColor } from '@patternfly/react-charts';
import { EventCallbackInterface, EventPropTypeInterface } from 'victory-core';

import { severityColors } from 'constants/visuals/colors';

const severityColorScale = Object.values(severityColors);

// Clone default PatternFly chart themes
const defaultTheme = getTheme(ChartThemeColor.multi);

export const defaultChartHeight = 280;

export const defaultChartBarWidth = 18;

/** A Victory chart theme based on grey/yellow/orange/red colors to indicate severity */
export const patternflySeverityTheme = {
    ...defaultTheme,
    stack: {
        ...defaultTheme.stack,
        colorScale: severityColorScale,
    },
    legend: {
        ...defaultTheme.legend,
        colorScale: severityColorScale,
    },
};

/**
 * A helper function to generate a chart onClick event that initiates navigation to another page.
 */
export function navigateOnClickEvent(
    history: History,
    /** A function that generates the link to navigate to when the entity is clicked */
    linkWith: (props: any) => string,
    /** An array of Victory onClick event handlers that will be called before navigation is initiated */
    defaultOnClicks: EventCallbackInterface<string, string>[] = []
): EventPropTypeInterface<'data', string | number | number[] | string[]> {
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
