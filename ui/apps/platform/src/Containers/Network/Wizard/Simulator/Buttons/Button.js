import React from 'react';
import PropTypes from 'prop-types';

const Button = ({ dataTestId, icon, text, onClick, disabled }) => {
    return (
        <button
            type="button"
            className="inline-block flex items-center my-3 px-3 text-center bg-primary-600 font-700 rounded-sm text-base-100 h-9 hover:bg-primary-700"
            onClick={onClick}
            disabled={disabled}
            dataTestId={dataTestId}
        >
            {icon}
            {text}
        </button>
    );
};

Button.propTypes = {
    dataTestId: PropTypes.string.isRequired,
    icon: PropTypes.element,
    text: PropTypes.string.isRequired,
    onClick: PropTypes.func.isRequired,
    disabled: PropTypes.bool,
};

Button.defaultProps = {
    icon: null,
    disabled: false,
};

export default Button;
