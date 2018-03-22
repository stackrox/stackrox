import React from 'react';
import PropTypes from 'prop-types';

const buttonClass =
    'flex py-2 px-2 rounded-sm font-400 uppercase text-center text-sm items-center ml-2';

const getButtonClass = type => {
    switch (type) {
        case 'prev':
            return `${buttonClass} text-primary-500 hover:text-white bg-white hover:bg-primary-400 border border-primary-400`;
        case 'next':
        case 'save':
            return `${buttonClass} text-success-500 hover:text-white bg-white hover:bg-success-400 border border-success-400`;
        default:
            return buttonClass;
    }
};

const SliderButton = props => (
    <button type="button" className={getButtonClass(props.type)} onClick={props.onClick}>
        {props.children}
    </button>
);

SliderButton.propTypes = {
    children: PropTypes.string.isRequired,
    onClick: PropTypes.func.isRequired,
    type: PropTypes.oneOf(['prev', 'next', 'save']).isRequired
};

export default SliderButton;
