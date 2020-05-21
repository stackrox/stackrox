import React from 'react';
import PropTypes from 'prop-types';

function DeploytimeMessages({ violations }) {
    return violations.map(({ message }) => (
        <div
            key={message}
            className="mb-4 p-3 pb-2 shadow border border-base-200 text-base-600 flex justify-between leading-normal bg-base-100"
        >
            {message}
        </div>
    ));
}

DeploytimeMessages.propTypes = {
    violations: PropTypes.arrayOf(
        PropTypes.shape({
            message: PropTypes.string.isRequired,
        })
    ),
};

DeploytimeMessages.defaultProps = {
    violations: [],
};

export default DeploytimeMessages;
