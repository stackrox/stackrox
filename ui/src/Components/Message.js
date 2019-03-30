import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

function Message(props) {
    const messageClasses = {
        warn:
            'warn-message p-4 rounded-sm text-warning-800 items-center border border-warning-700 bg-warning-200 leading-normal flex-no-shrink w-full',
        error:
            'error-message p-4 rounded-sm text-alert-800 items-center border border-alert-700 bg-alert-200 leading-normal flex-no-shrink w-full',
        info:
            'info-message p-4 rounded-sm text-success-800 items-center border border-success-700 bg-success-200 leading-normal flex-no-shrink w-full'
    };

    const borderColor = {
        warn: 'border-warning-300',
        error: 'border-alert-300',
        info: 'border-info-300'
    };

    const icons = {
        warn: <Icon.AlertTriangle className="h-6 w-6" strokeWidth="2px" />,
        error: <Icon.AlertTriangle className="h-6 w-6" strokeWidth="2px" />,
        info: <Icon.Check className="h-6 w-6" strokeWidth="2px" />
    };

    return (
        <div className={`flex ${messageClasses[props.type]}`}>
            <div
                className={`flex items-center justify-start flex-no-shrink pr-4 border-r ${
                    borderColor[props.type]
                }`}
            >
                <div className="flex p-4 rounded-full shadow-lg bg-base-100">
                    {icons[props.type]}
                </div>
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
