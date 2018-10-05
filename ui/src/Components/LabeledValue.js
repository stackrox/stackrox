import React from 'react';
import PropTypes from 'prop-types';

const LabeledValue = ({ label, value, valueClassName }) => (
    <div className="flex py-3" data-test-id="labeled-value">
        <div className="pr-1">{label}:</div>
        <div className={`flex-1 min-w-0 font-500 ${valueClassName}`}>{value}</div>
    </div>
);

LabeledValue.propTypes = {
    label: PropTypes.string.isRequired,
    value: PropTypes.string.isRequired,
    valueClassName: PropTypes.string
};

LabeledValue.defaultProps = {
    valueClassName: ''
};

export default LabeledValue;
