import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

const CloseButton = ({ className, iconColor, onClose }) => (
    <div
        className={`close-button relative flex items-end items-center cursor-pointer hover:bg-alert-300  ${className}`}
    >
        <span>
            <button
                type="button"
                className={`flex p-3 text-center text-sm items-center ${iconColor}`}
                onClick={onClose}
                aria-label="Close"
            >
                <Icon.X className="h-7 w-7" height={null} width={null} />
            </button>
        </span>
    </div>
);
CloseButton.propTypes = {
    onClose: PropTypes.func.isRequired,
    className: PropTypes.string,
    iconColor: PropTypes.string,
};

CloseButton.defaultProps = {
    className: '',
    iconColor: '',
};

export default CloseButton;
