import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

function Message(props) {
    const messageClasses = {
        warn:
            'warn-message p-4 rounded-sm text-warning-700 items-center border-2 border-base-100 bg-base-200 leading-normal flex-no-shrink',
        error:
            'error-message p-4 rounded-sm text-alert-800 text-sm items-center border-2 border-base-100 bg-alert-200 leading-normal flex-no-shrink',
        info:
            'info-message p-4 rounded-sm text-success-800 text-sm items-center border-2 border-base-100 bg-success-300 leading-normal flex-no-shrink'
    };

    const icons = {
        warn: <Icon.AlertTriangle className="h-10 w-10" strokeWidth="2px" />,
        error: <Icon.AlertOctagon className="h-4 w-4" strokeWidth="2px" />,
        info: <Icon.Check className="h-4 w-4" strokeWidth="2px" />
    };
    return (
        <div className={`flex ${messageClasses[props.type]}`}>
            <div className="h-8 w-8 self-center rounded-full flex items-center justify-center flex-no-shrink">
                {icons[props.type]}
            </div>
            <div className="flex pl-3 flex-1">{props.message}</div>
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
