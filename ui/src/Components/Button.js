import React from 'react';
import PropTypes from 'prop-types';
import { ClipLoader } from 'react-spinners';

const Loader = () => <ClipLoader loading size={20} color="currentColor" />;

const Button = ({
    className,
    icon,
    text,
    textCondensed,
    textClass,
    onClick,
    disabled,
    isLoading
}) => {
    const content = (
        <div className="flex">
            {icon}
            {textCondensed ? (
                <>
                    <span className={`${textClass} lg:hidden`}> {textCondensed} </span>
                    <span className="hidden lg:block"> {text} </span>
                </>
            ) : (
                <>
                    <span> {text} </span>
                </>
            )}
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
    textCondensed: PropTypes.string,
    textClass: PropTypes.string,
    onClick: PropTypes.func.isRequired,
    disabled: PropTypes.bool,
    isLoading: PropTypes.bool
};

Button.defaultProps = {
    icon: null,
    text: null,
    textCondensed: null,
    textClass: null,
    disabled: false,
    isLoading: false
};

export default Button;
