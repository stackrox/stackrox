import React from 'react';
import PropTypes from 'prop-types';
import * as Icons from 'react-feather';

const ErrorMessage = ({ message }) => (
    <div className="flex h-full items-center justify-center p-2">
        <Icons.XSquare size="48" />
        <div className="p-2 text-lg">
            <p>{message}</p>
        </div>
    </div>
);

ErrorMessage.propTypes = {
    message: PropTypes.string
};

ErrorMessage.defaultProps = {
    message: "We're sorry — something's gone wrong. The error has been logged."
};

export default ErrorMessage;
