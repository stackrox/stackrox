import React from 'react';
import { Link } from 'react-router-dom';
import { ChartLabel, ChartLabelProps } from '@patternfly/react-charts';

export type LinkableChartLabelProps = ChartLabelProps & {
    /**
     * Function that takes the `props` object passed to `ChartLabel` and
     * uses it to generate a link to navigate to when clicked.
     */
    linkWith: (props: ChartLabelProps) => string;
};

/**
 * Component that wraps a PatternFly `ChartLabel` component with a `Link` component
 * in order to use labels as links.
 */
export function LinkableChartLabel({ linkWith, ...props }: LinkableChartLabelProps) {
    return (
        <Link to={linkWith(props)}>
            <ChartLabel {...props} style={{ fill: 'var(--pf-global--link--Color)' }} />
        </Link>
    );
}
