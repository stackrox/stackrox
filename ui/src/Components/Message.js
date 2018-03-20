import React from 'react';
import PropTypes from 'prop-types';

function Message(props) {
    const messageClasses = {
        warn:
            'warn-message flex p-3 rounded-sm text-low-500 items-center m-3 bg-low-100 border-2 border-low-300 leading-normal',
        error:
            'error-message flex p-3 rounded-sm text-danger-600 text-sm items-center m-3 bg-danger-100 border-2 border-danger-400',
        info:
            'info-message flex p-3 rounded-sm text-success-600 text-sm items-center m-3 bg-success-100 border-2 border-success-400'
    };
    return <div className={messageClasses[props.type]}>{props.message}</div>;
}

Message.defaultProps = {
    type: 'info'
};

Message.propTypes = {
    message: PropTypes.string.isRequired,
    type: PropTypes.oneOf(['warn', 'error', 'info'])
};

export default Message;
