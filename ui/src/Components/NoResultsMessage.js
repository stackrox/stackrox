import React from 'react';
import PropTypes from 'prop-types';

const NoResultsMessage = props => (
    <div className="flex flex-1 items-center justify-center w-full leading-loose text-center h-full">
        {props.message}
    </div>
);

NoResultsMessage.propTypes = {
    message: PropTypes.string
};

NoResultsMessage.defaultProps = {
    message: 'No data available. Please ensure your cluster is properly configured.'
};

export default NoResultsMessage;
