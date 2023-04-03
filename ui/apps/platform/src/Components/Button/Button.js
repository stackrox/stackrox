import React from 'react';
import PropTypes from 'prop-types';
import { ClipLoader } from 'react-spinners';

const Loader = ({ size }) => <ClipLoader loading size={size} color="currentColor" />;

Loader.propTypes = {
    size: PropTypes.number.isRequired,
};

const Button = ({
    dataTestId,
    className,
    icon,
    text,
    textCondensed,
    textClass,
    onClick,
    disabled,
    isLoading,
    loaderSize,
    tabIndex,
    ...ariaProps
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
                text
            )}
        </div>
    );
    return (
        <button
            type="button"
            className={className}
            onClick={onClick}
            disabled={disabled}
            data-testid={dataTestId}
            tabIndex={tabIndex}
            {...ariaProps}
        >
            {isLoading ? <Loader size={loaderSize} /> : content}
        </button>
    );
};

Button.propTypes = {
    dataTestId: PropTypes.string,
    className: PropTypes.string,
    icon: PropTypes.element,
    text: PropTypes.string,
    textCondensed: PropTypes.string,
    textClass: PropTypes.string,
    onClick: PropTypes.func,
    disabled: PropTypes.bool,
    isLoading: PropTypes.bool,
    loaderSize: PropTypes.number,
    tabIndex: PropTypes.string,
};

Button.defaultProps = {
    dataTestId: null,
    className: '',
    icon: null,
    text: null,
    textCondensed: null,
    textClass: null,
    onClick: () => {},
    disabled: false,
    isLoading: false,
    loaderSize: 20,
    tabIndex: null,
};

export default Button;
