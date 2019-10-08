import React from 'react';
import PropTypes from 'prop-types';
import PercentageStackedPill from 'Components/visuals/PercentageStackedPill';

const getPercentageValue = (value, total) => Math.round((value / total) * 100);

const SeverityStackedPill = ({ low, medium, high, critical, tooltip }) => {
    const total = low + medium + high + critical;
    const data = [];
    if (low) {
        data.push({
            colorType: 'base',
            value: getPercentageValue(low, total)
        });
    }
    if (medium) {
        data.push({
            colorType: 'warning',
            value: getPercentageValue(medium, total)
        });
    }
    if (high) {
        data.push({
            colorType: 'caution',
            value: getPercentageValue(high, total)
        });
    }
    if (critical) {
        data.push({
            colorType: 'alert',
            value: getPercentageValue(critical, total)
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
        body: PropTypes.string.isRequired
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
