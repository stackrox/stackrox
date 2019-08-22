import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

const icons = {
    info: <Icon.CheckCircle className="h-8 w-8 mr-4 text-success-500" />
};

const NoResultsMessage = props => (
    <div
        className={`flex flex-1 items-center bg-base-100 justify-center w-full leading-loose text-center h-full ${
            props.className
        }`}
    >
        {props.icon && icons[props.icon]}
        {props.message}
    </div>
);

NoResultsMessage.propTypes = {
    message: PropTypes.string.isRequired,
    className: PropTypes.string,
    icon: PropTypes.string
};

NoResultsMessage.defaultProps = {
    className: '',
    icon: null
};

export default NoResultsMessage;
