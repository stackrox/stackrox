import React from 'react';
import PropTypes from 'prop-types';

const buttonClass =
    'flex py-2 px-2 rounded-sm font-400 uppercase text-center text-sm items-center ml-2 w-24 justify-center';

const getButtonClass = type => {
    switch (type) {
        case 'prev':
            return `${buttonClass} text-base-500 hover:text-white bg-white hover:bg-base-400 border border-base-400`;
        case 'next':
            return `${buttonClass} text-primary-500 hover:text-white bg-white hover:bg-primary-400 border border-primary-400`;
        case 'finish':
            return `${buttonClass} text-success-500 hover:text-white bg-white hover:bg-success-400 border border-success-400`;
        default:
            return buttonClass;
    }
};

const SliderButton = props => (
    <button
        type="button"
        className={getButtonClass(props.type)}
        onClick={props.onClick}
        disabled={props.disabled}
    >
        {props.children}
    </button>
);

SliderButton.propTypes = {
    children: PropTypes.oneOfType([PropTypes.string, PropTypes.element]).isRequired,
    onClick: PropTypes.func,
    type: PropTypes.oneOf(['prev', 'next', 'finish']).isRequired,
    disabled: PropTypes.bool
};

SliderButton.defaultProps = {
    disabled: false,
    onClick: null
};

export default SliderButton;
