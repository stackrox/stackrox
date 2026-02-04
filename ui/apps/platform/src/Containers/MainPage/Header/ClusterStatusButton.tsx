import type { CSSProperties, ReactElement } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import { Button, Flex, FlexItem, Tooltip } from '@patternfly/react-core';
import {
    CheckCircleIcon,
    ExclamationCircleIcon,
    ExclamationTriangleIcon,
    PortIcon,
} from '@patternfly/react-icons';

import { clustersBasePath } from 'routePaths';

const thClassName = 'font-400 pf-v5-u-pr-md pf-v5-u-text-align-left';
const tdClassName = 'pf-v5-u-text-align-right';

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
    const navigate = useNavigate();
    const hasDegradedClusters = degraded > 0;
    const hasUnhealthyClusters = unhealthy > 0;

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

    const onClick = () => {
        navigate(
            `${clustersBasePath}?s[Cluster status][0]=UNHEALTHY&s[Cluster status][1]=DEGRADED`
        );
    };

    // On masthead, black text on white background like a dropdown menu.
    const styleTooltip = {
        '--pf-v5-c-tooltip__content--Color': 'var(--pf-v5-global--Color--100)',
        '--pf-v5-c-tooltip__content--BackgroundColor': 'var(--pf-v5-global--BackgroundColor--100)',
    } as CSSProperties;

    // Using aria-label for accessibility instead of title to avoid two tooltips.
    return (
        <Tooltip
            content={contentElement}
            isContentLeftAligned
            position="bottom"
            style={styleTooltip}
        >
            <Button variant="plain" aria-label="Cluster status problems" onClick={onClick}>
                <Flex
                    direction={{ default: 'row' }}
                    flexWrap={{ default: 'nowrap' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                >
                    <FlexItem>
                        <PortIcon />
                    </FlexItem>
                    <FlexItem>
                        {hasUnhealthyClusters ? (
                            <ExclamationCircleIcon />
                        ) : hasDegradedClusters ? (
                            <ExclamationTriangleIcon />
                        ) : (
                            <CheckCircleIcon />
                        )}
                    </FlexItem>
                </Flex>
            </Button>
        </Tooltip>
    );
};

export default ClusterStatusButton;
