import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

const arrowButtonClass = 'absolute shadow bg-primary-100 h-16 cursor-pointer';
const arrowIconClass = 'h-8 w-8 text-primary-600 ';
const arrowStyles = {
    transform: 'translateY(-50%)',
    top: '50%'
};
const arrowPropTypes = {
    className: PropTypes.string,
    style: PropTypes.shape({}),
    onClick: PropTypes.func
};

const arrowDefaultProps = {
    className: '',
    style: {},
    onClick: null
};

const isArrowDisabled = className => className.includes('slick-disabled');

const NextArrow = props => {
    const { className, style, onClick } = props;
    const isDisabled = isArrowDisabled(className);
    return (
        <div
            className={`${className} absolute z-10 pin-r h-full pointer-events-none ${isDisabled &&
                'hidden'}`}
        >
            <button
                type="button"
                style={{ ...style, ...arrowStyles }}
                className={`${arrowButtonClass} pin-r rounded-l-full hover:bg-secondary-200 pointer-events-auto`}
                onClick={onClick}
            >
                <Icon.ChevronRight className={`${arrowIconClass} ml-3`} />
            </button>
        </div>
    );
};

NextArrow.propTypes = arrowPropTypes;
NextArrow.defaultProps = arrowDefaultProps;

const PrevArrow = props => {
    const { className, style, onClick } = props;
    const isDisabled = isArrowDisabled(className);
    return (
        <div
            style={{ ...style, ...arrowStyles }}
            className={`${className} absolute z-10 pin-l h-full pointer-events-none ${isDisabled &&
                'hidden'}`}
        >
            <button
                type="button"
                style={{ ...style, ...arrowStyles }}
                className={`${arrowButtonClass} pin-l rounded-r-full hover:bg-secondary-200 pointer-events-auto`}
                onClick={onClick}
            >
                <Icon.ChevronLeft className={`${arrowIconClass} mr-3`} />
            </button>
        </div>
    );
};

PrevArrow.propTypes = arrowPropTypes;
PrevArrow.defaultProps = arrowDefaultProps;

export { NextArrow, PrevArrow };
