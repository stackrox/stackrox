import React from 'react';
import PropTypes from 'prop-types';
import { ClipLoader } from 'react-spinners';

const Loader = () => <ClipLoader loading size={20} color="#5E667D" />;

const Button = ({ className, icon, text, onClick, disabled, isLoading }) => {
    const content = (
        <div className="flex">
            {icon} {text}
        </div>
    );
    return (
        <button type="button" className={className} onClick={onClick} disabled={disabled}>
            {isLoading ? <Loader /> : content}
        </button>
    );
};

Button.propTypes = {
    className: PropTypes.string.isRequired,
    icon: PropTypes.element,
    text: PropTypes.string,
    onClick: PropTypes.func.isRequired,
    disabled: PropTypes.bool,
    isLoading: PropTypes.bool
};

Button.defaultProps = {
    icon: null,
    text: null,
    disabled: false,
    isLoading: false
};

export default Button;
