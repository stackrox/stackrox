import React from 'react';
import PropTypes from 'prop-types';

import PercentageStackedPill from 'Components/visuals/PercentageStackedPill';
import { getPercentage } from 'utils/mathUtils';

const SeverityStackedPill = ({ low, moderate, important, critical, tooltip }) => {
    const total = low + moderate + important + critical;
    const data = [];
    if (low) {
        data.push({
            colorType: 'base',
            value: getPercentage(low, total),
        });
    }
    if (moderate) {
        data.push({
            colorType: 'warning',
            value: getPercentage(moderate, total),
        });
    }
    if (important) {
        data.push({
            colorType: 'caution',
            value: getPercentage(important, total),
        });
    }
    if (critical) {
        data.push({
            colorType: 'alert',
            value: getPercentage(critical, total),
        });
    }
    return <PercentageStackedPill data={data} tooltip={tooltip} />;
};

SeverityStackedPill.propTypes = {
    low: PropTypes.number,
    moderate: PropTypes.number,
    important: PropTypes.number,
    critical: PropTypes.number,
    tooltip: PropTypes.shape({
        title: PropTypes.string.isRequired,
        body: PropTypes.oneOfType([PropTypes.string, PropTypes.element]).isRequired,
    }),
};

SeverityStackedPill.defaultProps = {
    low: 0,
    moderate: 0,
    important: 0,
    critical: 0,
    tooltip: null,
};

export default SeverityStackedPill;
