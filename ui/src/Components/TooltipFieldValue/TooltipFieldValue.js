import React from 'react';
import PropTypes from 'prop-types';

const types = ['alert', 'caution', 'warning'];

const TooltipFieldValue = ({ dataTestId, field, value, type }) => {
    if (value === null) return null;
    const textColor = types.includes(type) ? `text-${type}-600` : '';
    return (
        <div className={textColor} data-testid={dataTestId}>
            <span className="font-700">{field}: </span>
            <span>{value}</span>
        </div>
    );
};

TooltipFieldValue.propTypes = {
    dataTestId: PropTypes.string,
    field: PropTypes.string.isRequired,
    value: PropTypes.oneOfType([PropTypes.number, PropTypes.string]),
    type: PropTypes.oneOf(types),
};

TooltipFieldValue.defaultProps = {
    dataTestId: null,
    value: null,
    type: null,
};

export default TooltipFieldValue;
