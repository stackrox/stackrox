import React from 'react';
import PropTypes from 'prop-types';

import PercentageStackedPill from 'Components/visuals/PercentageStackedPill';
import { getPercentage } from 'utils/mathUtils';

const SeverityStackedPill = ({ low, medium, high, critical, tooltip }) => {
    const total = low + medium + high + critical;
    const data = [];
    if (low) {
        data.push({
            colorType: 'base',
            value: getPercentage(low, total)
        });
    }
    if (medium) {
        data.push({
            colorType: 'warning',
            value: getPercentage(medium, total)
        });
    }
    if (high) {
        data.push({
            colorType: 'caution',
            value: getPercentage(high, total)
        });
    }
    if (critical) {
        data.push({
            colorType: 'alert',
            value: getPercentage(critical, total)
        });
    }
    return <PercentageStackedPill data={data} tooltip={tooltip} />;
};

SeverityStackedPill.propTypes = {
    low: PropTypes.number,
    medium: PropTypes.number,
    high: PropTypes.number,
    critical: PropTypes.number,
    tooltip: PropTypes.shape({
        title: PropTypes.string.isRequired,
        body: PropTypes.oneOfType([PropTypes.string, PropTypes.element]).isRequired
    })
};

SeverityStackedPill.defaultProps = {
    low: 0,
    medium: 0,
    high: 0,
    critical: 0,
    tooltip: null
};

export default SeverityStackedPill;
