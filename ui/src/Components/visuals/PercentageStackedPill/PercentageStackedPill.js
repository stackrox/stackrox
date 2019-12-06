import React from 'react';
import PropTypes from 'prop-types';
import { colorTypes, defaultColorType } from 'constants/visuals/colors';

import Tooltip from 'rc-tooltip';

const TooltipOverlay = ({ title, body }) => {
    if (!title || !body) return null;
    return (
        <div className="">
            <h2 className="border-b border-primary-400 mb-1 pb-1 graph-hint-title text-sm">
                {title}
            </h2>
            <div className="graph-hint-body py-1 text-xs">{body}</div>
        </div>
    );
};

const getBackgroundColor = colorType => {
    const color = colorTypes.find(datum => datum === colorType);
    if (!color) return defaultColorType;
    return `bg-${color}-400`;
};

const PercentageStackedPill = ({ data, tooltip }) => {
    const pills = data.map(({ value, colorType }, i, arr) => {
        let className = `border-r border-base-100 ${getBackgroundColor(colorType)}`;
        // adds a rounded corner to the left-most pill
        if (i === 0) className = `${className} rounded-l-full`;
        // adds a rounded corner to the right-most pill
        if (i === arr.length - 1) className = `${className} rounded-r-full`;

        return <div className={className} key={i} style={{ width: `${value}%` }} />;
    });
    const { title: tooltipTitle, body: tooltipBody } = tooltip || {};
    return (
        <Tooltip
            placement="top"
            overlay={<TooltipOverlay title={tooltipTitle} body={tooltipBody} />}
            mouseLeaveDelay={0}
            overlayClassName="opacity-100"
        >
            <div
                className="flex rounded-full w-full min-w-10 max-w-24 h-3 border border-base-300 bg-base-200"
                style={{ boxShadow: 'inset 0 0px 8px 0 hsla(0, 0%, 0%, .10) !important' }}
            >
                {pills}
            </div>
        </Tooltip>
    );
};

PercentageStackedPill.propTypes = {
    data: PropTypes.arrayOf(
        PropTypes.shape({
            colorType: PropTypes.string.isRequired,
            value: PropTypes.number.isRequired
        })
    ),
    tooltip: PropTypes.shape({
        title: PropTypes.string.isRequired,
        body: PropTypes.oneOfType([PropTypes.string, PropTypes.element]).isRequired
    })
};

PercentageStackedPill.defaultProps = {
    data: [],
    tooltip: null
};

export default PercentageStackedPill;
