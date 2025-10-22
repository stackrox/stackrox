import React from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { ChartLabel } from '@patternfly/react-charts';
import type { ChartLabelProps } from '@patternfly/react-charts';

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
            <ChartLabel {...props} style={{ fill: 'var(--pf-v5-global--link--Color)' }} />
        </Link>
    );
}
