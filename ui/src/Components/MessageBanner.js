import React, { useState } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

const className = 'w-full flex items-center justify-center leading-normal p-3';
const messageClasses = {
    warn: `${className} bg-warning-300 text-warning-800`,
    error: `${className} bg-alert-300 text-alert-800`,
    info: `${className} bg-info-300 text-info-800`
};

function MessageBanner({ message, type, showCancel, onCancel }) {
    const [isBannerShowing, showBanner] = useState(true);
    function onClickHandler() {
        showBanner(false);
        if (onCancel) onCancel();
    }
    return (
        isBannerShowing && (
            <div className={messageClasses[type]}>
                <div className="flex flex-1 justify-center">{message}</div>
                {showCancel && (
                    <Icon.X className="h-6 w-6 cursor-pointer" onClick={onClickHandler} />
                )}
            </div>
        )
    );
}

MessageBanner.defaultProps = {
    type: 'info',
    showCancel: false,
    onCancel: null
};

MessageBanner.propTypes = {
    message: PropTypes.string.isRequired,
    type: PropTypes.oneOf(['warn', 'error', 'info']),
    showCancel: PropTypes.bool,
    onCancel: PropTypes.func
};

export default MessageBanner;
