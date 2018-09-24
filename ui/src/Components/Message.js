import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

function Message(props) {
    const messageClasses = {
        warn: 'warn-message p-3 rounded-sm text-low-500 items-center m-3 bg-low-100 leading-normal',
        error:
            'error-message p-3 rounded-sm text-danger-600 text-sm items-center m-3 bg-danger-100 leading-normal',
        info:
            'info-message p-3 rounded-sm text-success-600 text-sm items-center m-3 bg-success-100 leading-normal'
    };

    const icons = {
        warn: <Icon.AlertTriangle className="h-10 w-10" strokeWidth="1.5px" />,
        error: <Icon.X className="h-4 w-4" strokeWidth="1.5px" />,
        info: <Icon.Check className="h-4 w-4" strokeWidth="1.5px" />
    };
    return (
        <div className={`flex flex-row ${messageClasses[props.type]}`}>
            <div className="h-8 w-8 self-center rounded-full flex items-center justify-center bg-white flex-no-shrink">
                {icons[props.type]}
            </div>
            <div className="flex pl-5 flex-1">{props.message}</div>
        </div>
    );
}

Message.defaultProps = {
    type: 'info'
};

Message.propTypes = {
    message: PropTypes.string.isRequired,
    type: PropTypes.oneOf(['warn', 'error', 'info'])
};

export default Message;
