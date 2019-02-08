import React from 'react';
import PropTypes from 'prop-types';

const NoResultsMessage = props => (
    <div className="flex flex-col h-full border-r border-base-400 min-w-0 w-full justify-center items-center">
        <div className="text-warning-800 bg-warning-200 border-2 border-warning-300 p-6 rounded">
            {props.message}
        </div>
    </div>
);

NoResultsMessage.propTypes = {
    message: PropTypes.string
};

NoResultsMessage.defaultProps = {
    message: 'No data available. Please ensure your cluster is properly configured.'
};

export default NoResultsMessage;
