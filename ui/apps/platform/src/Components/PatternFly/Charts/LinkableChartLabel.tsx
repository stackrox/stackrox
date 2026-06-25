import { Link } from 'react-router-dom-v5-compat';
import { ChartLabel } from '@patternfly/react-charts/victory';
import type { ChartLabelProps } from '@patternfly/react-charts/victory';

export type LinkableChartLabelProps = ChartLabelProps & {
    /**
     * Function that takes the `props` object passed to `ChartLabel` and
     * uses it to generate a link to navigate to when clicked.
     */
    linkWith: (props: ChartLabelProps) => string;
    onClick?: (props: ChartLabelProps) => void;
};

/**
 * Component that wraps a PatternFly `ChartLabel` component with a `Link` component
 * in order to use labels as links.
 */
export function LinkableChartLabel({ linkWith, onClick, ...props }: LinkableChartLabelProps) {
    return (
        <Link to={linkWith(props)} onClick={() => onClick?.(props)}>
            <ChartLabel
                {...props}
                style={{
                    fill: 'var(--pf-t--global--text--color--link--default)',
                }}
            />
        </Link>
    );
}
