import React from 'react';
import PropTypes from 'prop-types';
import { ClipLoader } from 'react-spinners';

const Loader = ({ size }) => <ClipLoader loading size={size} color="currentColor" />;

Loader.propTypes = {
    size: PropTypes.number.isRequired
};

const Button = ({
    className,
    icon,
    text,
    textCondensed,
    textClass,
    onClick,
    disabled,
    isLoading,
    loaderSize
}) => {
    const content = (
        <div className="flex items-center">
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
            {isLoading ? <Loader size={loaderSize} /> : content}
        </button>
    );
};

Button.propTypes = {
    className: PropTypes.string,
    icon: PropTypes.element,
    text: PropTypes.string,
    textCondensed: PropTypes.string,
    textClass: PropTypes.string,
    onClick: PropTypes.func.isRequired,
    disabled: PropTypes.bool,
    isLoading: PropTypes.bool,
    loaderSize: PropTypes.number
};

Button.defaultProps = {
    className: '',
    icon: null,
    text: null,
    textCondensed: null,
    textClass: null,
    disabled: false,
    isLoading: false,
    loaderSize: 20
};

export default Button;
