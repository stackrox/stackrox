import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';
import 'rc-tooltip/assets/bootstrap.css';

const CloseButton = ({ onClose }) => (
    <div className="flex items-end border-base-300 items-center hover:bg-primary-300 ml-2 border-l">
        <span>
            <Tooltip placement="top" overlay={<div>Cancel</div>}>
                <button
                    className="flex text-primary-600 p-4 text-center text-sm items-center hover:text-white"
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
    onClose: PropTypes.func.isRequired
};

const Panel = ({ header, buttons, className, onClose, children }) => (
    <div
        className={`flex flex-col bg-white border h-full border-t-0 border-base-300 ${className}`}
        data-test-id="panel"
    >
        <div className="shadow-underline font-bold bg-primary-100">
            <div className="flex flex-row w-full">
                <div
                    className="flex flex-1 text-base-600 uppercase items-center tracking-wide py-2 px-4"
                    data-test-id="panel-header"
                >
                    {header}
                </div>
                <div className="flex items-center py-2 px-4">{buttons}</div>
                {onClose && <CloseButton onClose={onClose} />}
            </div>
        </div>
        <div className="flex flex-1 overflow-auto transition">{children}</div>
    </div>
);

Panel.propTypes = {
    header: PropTypes.string,
    buttons: PropTypes.node,
    className: PropTypes.string,
    children: PropTypes.node.isRequired,
    onClose: PropTypes.func
};

Panel.defaultProps = {
    header: ' ',
    buttons: null,
    className: 'w-full',
    onClose: null
};

export default Panel;
