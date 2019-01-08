import React from 'react';
import PropTypes from 'prop-types';

const Button = ({ className, icon, text, onClick, disabled }) => (
    <button type="button" className={className} onClick={onClick} disabled={disabled}>
        {icon} {text}
    </button>
);

Button.propTypes = {
    className: PropTypes.string.isRequired,
    icon: PropTypes.element,
    text: PropTypes.string,
    onClick: PropTypes.func.isRequired,
    disabled: PropTypes.bool
};

Button.defaultProps = {
    icon: null,
    text: null,
    disabled: false
};

export default Button;
