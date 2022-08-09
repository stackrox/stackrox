import React, { CSSProperties, ReactElement } from 'react';
import { Activity } from 'react-feather';
import { useHistory } from 'react-router-dom';
import { Tooltip } from '@patternfly/react-core';

import { clustersBasePath } from 'routePaths';

const thClassName = 'font-400 pf-u-pr-md pf-u-text-align-left';
const tdClassName = 'pf-u-text-align-right';

type ClusterStatusButtonProps = {
    degraded?: number;
    unhealthy?: number;
};

/*
 * Visual indicator in top navigation whether any clusters have health problems.
 *
 * The tooltip body displays query results, including zero counts.
 * A button click opens the Clusters list.
 */
const ClusterStatusButton = ({
    degraded = 0,
    unhealthy = 0,
}: ClusterStatusButtonProps): ReactElement => {
    const history = useHistory();
    const hasDegradedClusters = degraded > 0;
    const hasUnhealthyClusters = unhealthy > 0;
    const hasProblems = hasDegradedClusters || hasUnhealthyClusters;

    const contentElement = (
        <div>
            <div>Cluster status problems</div>
            <table>
                <tbody>
                    <tr key="unhealthy">
                        <th className={thClassName} scope="row">
                            Unhealthy
                        </th>
                        <td className={tdClassName}>{unhealthy}</td>
                    </tr>
                    <tr key="degraded">
                        <th className={thClassName} scope="row">
                            Degraded
                        </th>
                        <td className={tdClassName}>{degraded}</td>
                    </tr>
                </tbody>
            </table>
        </div>
    );

    // Border radius for background circle to emphasize icon color.
    const classNameProblems = hasProblems ? 'rounded-lg' : '';

    let styleProblems: CSSProperties | undefined;
    /*
     * Explicit white background because the following did not work:
     * `backgroundColor: var(--pf-global--BackgroundColor--100)`
     */
    if (hasUnhealthyClusters) {
        styleProblems = {
            backgroundColor: '#ffffff',
            color: 'var(--pf-global--danger-color--100)',
        };
    } else if (hasDegradedClusters) {
        styleProblems = {
            backgroundColor: '#ffffff',
            color: 'var(--pf-global--warning-color--100)',
        };
    }

    const onClick = () => {
        history.push({
            pathname: clustersBasePath,
            search: '',
            // TODO after ClustersPage sets search filter according to search query string in URL:
            // If any clusters have problems, then Clusters list has search filter.
            // search: hasUnhealthyClusters || hasDegradedClusters ? '?s[Cluster Health][0]=UNHEALTHY&s[Cluster Health][1]=DEGRADED' : '',
        });
    };

    // On masthead, black text on white background like a dropdown menu.
    const styleTooltip = {
        '--pf-c-tooltip__content--Color': 'var(--pf-global--Color--100)',
        '--pf-c-tooltip__content--BackgroundColor': 'var(--pf-global--BackgroundColor--100)',
    } as CSSProperties;

    // Using aria-label for accessibility instead of title to avoid two tooltips.
    return (
        <Tooltip
            content={contentElement}
            isContentLeftAligned
            position="bottom"
            style={styleTooltip}
        >
            <button
                aria-label="Cluster status problems"
                type="button"
                onClick={onClick}
                className="flex h-full items-center pt-2 pb-2 px-4"
            >
                <div className={classNameProblems} style={styleProblems}>
                    <Activity className="h-4 w-4" />
                </div>
            </button>
        </Tooltip>
    );
};

export default ClusterStatusButton;
