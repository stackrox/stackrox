import React from 'react';
import PropTypes from 'prop-types';
import { Tooltip } from '@patternfly/react-core';

import DetailedTooltipContent from 'Components/DetailedTooltipContent';
import { colorTypes, defaultColorType } from 'constants/visuals/colors';

const getBackgroundColor = (colorType) => {
    const color = colorTypes.find((datum) => datum === colorType);
    if (!color) {
        return defaultColorType;
    }
    return `bg-${color}-500`;
};

const PercentageStackedPill = ({ data, tooltip }) => {
    const pills = data.map(({ value, colorType }, i, arr) => {
        let className = `border-r border-base-100 ${getBackgroundColor(colorType)}`;
        // adds a rounded corner to the left-most pill
        if (i === 0) {
            className = `${className} rounded-l-full`;
        }
        // adds a rounded corner to the right-most pill
        if (i === arr.length - 1) {
            className = `${className} rounded-r-full`;
        }

        // eslint-disable-next-line react/no-array-index-key
        return <div className={className} key={i} style={{ width: `${value}%` }} />;
    });
    const { title: tooltipTitle, body: tooltipBody } = tooltip || {};
    const content = (
        <div
            className="flex rounded-full w-full min-w-10 max-w-24 h-3 bg-base-300"
            style={{ boxShadow: 'inset 0 0px 8px 0 hsla(0, 0%, 0%, .10) !important' }}
        >
            {pills}
        </div>
    );
    return tooltip ? (
        <Tooltip
            isContentLeftAligned
            content={<DetailedTooltipContent title={tooltipTitle} body={tooltipBody} />}
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
            value: PropTypes.number.isRequired,
        })
    ),
    tooltip: PropTypes.shape({
        title: PropTypes.string.isRequired,
        body: PropTypes.oneOfType([PropTypes.string, PropTypes.element]).isRequired,
    }),
};

PercentageStackedPill.defaultProps = {
    data: [],
    tooltip: null,
};

export default PercentageStackedPill;
