import React from 'react';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';

import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import { colorTypes, defaultColorType } from 'constants/visuals/colors';

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
    const content = (
        <div
            className="flex rounded-full w-full min-w-10 max-w-24 h-3 border border-base-300 bg-base-200"
            style={{ boxShadow: 'inset 0 0px 8px 0 hsla(0, 0%, 0%, .10) !important' }}
        >
            {pills}
        </div>
    );
    return tooltip ? (
        <Tooltip
            placement="top"
            overlay={<DetailedTooltipOverlay title={tooltipTitle} body={tooltipBody} />}
            mouseLeaveDelay={0}
            overlayClassName="opacity-100 p-0"
        >
            {content}
        </Tooltip>
    ) : (
        content
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
