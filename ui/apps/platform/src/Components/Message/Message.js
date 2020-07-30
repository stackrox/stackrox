import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

import Loader from 'Components/Loader';

function Message(props) {
    const messageClasses = {
        warn:
            'warn-message p-4 rounded-sm text-warning-800 items-center border border-warning-700 bg-warning-200 leading-normal flex-shrink-0 w-full',
        error:
            'error-message p-4 rounded-sm text-alert-800 items-center border border-alert-700 bg-alert-200 leading-normal flex-shrink-0 w-full',
        info:
            'info-message p-4 rounded-sm text-success-800 items-center border border-success-700 bg-success-200 leading-normal flex-shrink-0 w-full',
        guidance:
            'guidance-message p-4 rounded-sm text-primary-800 items-center border border-primary-700 bg-primary-200 leading-normal flex-shrink-0 w-full',
        loading:
            'loading-message p-4 rounded-sm text-primary-800 items-center border border-primary-700 bg-primary-200 leading-normal flex-shrink-0 w-full',
    };

    const borderColor = {
        warn: 'border-warning-300',
        error: 'border-alert-300',
        info: 'border-info-300',
        guidance: 'border-primary-300',
        loading: 'border-primary-300',
    };

    const icons = {
        warn: <Icon.AlertTriangle className="h-6 w-6" strokeWidth="2px" />,
        error: <Icon.AlertTriangle className="h-6 w-6" strokeWidth="2px" />,
        info: <Icon.Check className="h-6 w-6" strokeWidth="2px" />,
        guidance: <Icon.Info className="h-6 w-6" strokeWidth="2px" />,
        loading: <Loader message={null} />,
    };

    return (
        <div className={`flex ${messageClasses[props.type]}`} data-testid="message">
            <div
                className={`flex items-center justify-start flex-shrink-0 pr-4 border-r ${
                    borderColor[props.type]
                }`}
            >
                <div className="flex p-4 rounded-full shadow-lg bg-base-100">
                    {icons[props.type]}
                </div>
            </div>
            <div className="flex flex-grow pl-3">{props.message}</div>
        </div>
    );
}

Message.propTypes = {
    message: PropTypes.oneOfType([PropTypes.string, PropTypes.element]).isRequired,
    type: PropTypes.oneOf(['warn', 'error', 'info', 'guidance', 'loading']),
};

Message.defaultProps = {
    type: 'info',
};

export default Message;
