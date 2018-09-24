import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';

const CloseButton = ({ className, iconColor, onClose }) => (
    <div
        className={`flex items-end items-center ml-2 cursor-pointer border-base-300 border-l hover:bg-primary-300 ${className}`}
    >
        <span>
            <Tooltip placement="top" overlay={<div>Cancel</div>}>
                <button
                    className={`flex p-3 text-center text-sm items-center hover:text-white ${iconColor}`}
                    onClick={onClose}
                    data-test-id="cancel"
                >
                    <Icon.X className="h-4 w-4" />
                </button>
            </Tooltip>
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
