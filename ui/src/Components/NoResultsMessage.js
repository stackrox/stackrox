import React from 'react';
import PropTypes from 'prop-types';

const NoResultsMessage = props => (
    <div className="flex flex-1 items-center justify-center w-full leading-loose text-center font-700 h-full">
        {props.message}
    </div>
);

NoResultsMessage.propTypes = {
    message: PropTypes.string.isRequired
};

export default NoResultsMessage;
