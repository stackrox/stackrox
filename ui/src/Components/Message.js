import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

function Message(props) {
    const messageClasses = {
        warn:
            'warn-message p-4 rounded-sm text-warning-800 items-center border border-warning-700 bg-warning-200 leading-normal flex-no-shrink justify-center w-full',
        error:
            'error-message p-4 rounded-sm text-alert-800 items-center border border-alert-700 bg-alert-200 leading-normal flex-no-shrink justify-center w-full',
        info:
            'info-message p-4 rounded-sm text-success-800 items-center border border-success-700 bg-success-200 leading-normal flex-no-shrink justify-center w-full'
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
            <div className="flex pl-3">{props.message}</div>
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
