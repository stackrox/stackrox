/* eslint-disable react/jsx-no-bind */
import React from 'react';
import PropTypes from 'prop-types';
import { Activity } from 'react-feather';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';

import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import Tooltip from 'Components/Tooltip';
import { clustersPath } from 'routePaths';

const bgHoverDefault = 'hover:bg-base-200';
const fgColorDefault = 'text-base-600';

const bgHoverUnhealthy = 'hover:bg-alert-200';
const fgColorUnhealthy = 'text-alert-700';
const bothColorsUnhealthy = `bg-alert-200 ${fgColorUnhealthy}`;

const bgHoverDegraded = 'hover:bg-warning-200';
const fgColorDegraded = 'text-warning-700';

const trClassName = 'align-top leading-normal';
const thClassName = 'font-600 pl-0 pr-2 py-0 text-left';
const tdClassName = 'p-0 text-right';

/*
 * Visual indicator in top navigation whether any clusters have health problems.
 *
 * The right corners of the button display non-zero counts.
 * The tooltip body displays query results, including zero counts.
 * A button click opens the Clusters list with a search query.
 */
const ClusterStatusButton = ({ degraded, unhealthy, history }) => {
    const hasDegradedClusters = degraded > 0;
    const hasUnhealthyClusters = unhealthy > 0;

    // Use table instead of TooltipFieldValue to align numbers.
    // Because of flex-col, tooltip body has full width of tooltip,
    // therefore a div wrapper in needed so that its child table
    // can have automatic width as little as its content needs.
    const resultsElement = (
        <div>
            <table>
                <tbody>
                    <tr
                        className={
                            hasUnhealthyClusters
                                ? `${trClassName} ${bothColorsUnhealthy}`
                                : trClassName
                        }
                        key="unhealthy"
                    >
                        <th className={thClassName} scope="row">
                            Unhealthy
                        </th>
                        <td className={tdClassName}>{unhealthy}</td>
                    </tr>
                    <tr
                        className={
                            hasDegradedClusters ? `${trClassName} ${fgColorDegraded}` : trClassName
                        }
                        key="degraded"
                    >
                        <th className={thClassName} scope="row">
                            Degraded
                        </th>
                        <td className={tdClassName}>{degraded}</td>
                    </tr>
                </tbody>
            </table>
        </div>
    );

    // Unhealthy at upper right to suggest higher severity.
    const unhealthyElement = hasUnhealthyClusters ? (
        <span
            aria-label="Number of clusters with Unhealthy status"
            className={`absolute top-0 right-0 p-1 rounded-bl ${bothColorsUnhealthy}`}
        >
            {unhealthy}
        </span>
    ) : null;

    // Degraded at lower right to suggest lower severity.
    const degradedElement = hasDegradedClusters ? (
        <span
            aria-label="Number of clusters with Degraded status"
            className={`absolute bottom-0 right-0 p-1 ${fgColorDegraded}`}
        >
            {degraded}
        </span>
    ) : null;

    let bgHover = bgHoverDefault;
    let iconColor = fgColorDefault;

    // The color indicates the more severe health problem.
    if (hasUnhealthyClusters) {
        bgHover = bgHoverUnhealthy;
        iconColor = fgColorUnhealthy;
    } else if (hasDegradedClusters) {
        bgHover = bgHoverDegraded;
        iconColor = fgColorDegraded;
    }

    const onClick = () => {
        history.push({
            pathname: clustersPath.replace('/:clusterId?', ''),
            search: '',
            // TODO after ClustersPage sets search filter according to search query string in URL:
            // If any clusters have problems, then Clusters list has search filter.
            // search: hasUnhealthyClusters || hasDegradedClusters ? '?s[Cluster Health][0]=UNHEALTHY&s[Cluster Health][1]=DEGRADED' : '',
        });
    };

    // Using aria-label for accessibility instead of title to avoid two tooltips.
    // The tooltip has title and subtitle partly to limit its width to the minimum,
    // because the button is near the right edge of the top navigation bar.
    return (
        <Tooltip
            content={
                <DetailedTooltipOverlay
                    title="Cluster Status"
                    subtitle="Problems"
                    body={resultsElement}
                />
            }
        >
            <button
                aria-label="Cluster Status Problems"
                type="button"
                onClick={onClick}
                className={`relative flex font-600 h-full items-center px-4 border-base-400 border-l border-r-0 ${bgHover}`}
            >
                {unhealthyElement}
                {degradedElement}
                <span className={iconColor}>
                    <Activity className="h-4 w-4" />
                </span>
            </button>
        </Tooltip>
    );
};

ClusterStatusButton.propTypes = {
    degraded: PropTypes.number,
    unhealthy: PropTypes.number,
    history: ReactRouterPropTypes.history.isRequired,
};

ClusterStatusButton.defaultProps = {
    degraded: 0,
    unhealthy: 0,
};

export default withRouter(ClusterStatusButton);
