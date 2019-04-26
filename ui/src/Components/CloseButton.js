import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

const CloseButton = ({ className, iconColor, onClose }) => (
    <div
        className={`close-button relative flex items-end items-center lg:ml-2 cursor-pointer border-base-400 border-l hover:bg-primary-300 hover:border-primary-300 ${className}`}
    >
        <span>
            <button
                type="button"
                className={`flex p-3 text-center text-sm items-center ${iconColor}`}
                onClick={onClose}
                data-test-id="cancel"
            >
                <Icon.X className="h-7 w-7" />
            </button>
        </span>
    </div>
);
CloseButton.propTypes = {
    onClose: PropTypes.func.isRequired,
    className: PropTypes.string,
    iconColor: PropTypes.string
};

CloseButton.defaultProps = {
    className: '',
    iconColor: ''
};

export default CloseButton;
