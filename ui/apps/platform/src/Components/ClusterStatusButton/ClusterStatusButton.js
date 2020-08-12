import React from 'react';
import PropTypes from 'prop-types';
import { Activity } from 'react-feather';

import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import Tooltip from 'Components/Tooltip';

const bgHoverDefault = 'hover:bg-base-200';
const fgColorDefault = 'text-base-600';

const bgHoverUnhealthy = 'hover:bg-alert-200';
const bgColorUnhealthy = 'bg-alert-200';
const fgColorUnhealthy = 'text-alert-700';

const bgHoverDegraded = 'hover:bg-warning-200';
const bgColorDegraded = 'bg-warning-200';
const fgColorDegraded = 'text-warning-700';

const trClassName = 'align-top leading-normal';
const thClassName = 'font-600 p-0 text-left w-16';
const tdClassName = 'p-0 text-right';

/*
 * Visual indicator in top navigation whether any clusters have health problems.
 *
 * The right corners of the button display non-zero counts.
 * The tooltip body displays query results, including zero counts.
 * A button click opens the Clusters list with a search query.
 */
const ClusterStatusButton = ({ degraded, unhealthy }) => {
    const hasDegradedClusters = degraded > 0;
    const hasUnhealthyClusters = unhealthy > 0;

    // Use table instead of TooltipFieldValue to align numbers.
    // Each th has same colors as corresponding cells in Clusters list.
    // Each td has same colors as counts in right corners of this button.
    const resultsElement = (
        <table className="table-fixed w-24">
            <tbody>
                <tr className={`${trClassName} ${fgColorUnhealthy}`} key="unhealthy">
                    <th className={thClassName} scope="row">
                        <span className={bgColorUnhealthy}>Unhealthy</span>
                    </th>
                    <td className={tdClassName}>{unhealthy}</td>
                </tr>
                <tr className={`${trClassName} ${fgColorDegraded}`} key="degraded">
                    <th className={thClassName} scope="row">
                        <span className={bgColorDegraded}>Degraded</span>
                    </th>
                    <td className={tdClassName}>{degraded}</td>
                </tr>
            </tbody>
        </table>
    );

    // Unhealthy at upper right to suggest higher severity.
    const unhealthyElement = hasUnhealthyClusters ? (
        <span
            aria-label="Number of clusters with Unhealthy status"
            className={`absolute top-0 right-0 p-1 ${fgColorUnhealthy}`}
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

    // Using aria-label for accessibility instead of title to avoid two tooltips.
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
};

ClusterStatusButton.defaultProps = {
    degraded: 0,
    unhealthy: 0,
};

export default ClusterStatusButton;
