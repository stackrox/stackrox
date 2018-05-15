import React from 'react';
import PropTypes from 'prop-types';

const NoResultsMessage = props => (
    <div className="flex flex-1 items-center justify-center w-full h-full">{props.message}</div>
);

NoResultsMessage.propTypes = {
    message: PropTypes.string.isRequired
};

export default NoResultsMessage;
