import React from 'react';
import PropTypes from 'prop-types';
import 'rc-tooltip/assets/bootstrap.css';
import CloseButton from './CloseButton';

const Panel = props => (
    <div
        className={`flex flex-col bg-white border h-full border-t-0 border-base-300 ${
            props.className
        }`}
        data-test-id="panel"
    >
        <div className="shadow-underline font-bold bg-white">
            <div className="flex flex-row w-full py-1">
                <div
                    className="flex flex-1 text-base-600 uppercase items-center tracking-wide py-2 px-4"
                    data-test-id="panel-header"
                >
                    {props.header}
                </div>
                <div className="flex items-center py-2 px-4">{props.buttons}</div>
                {props.headerComponents && (
                    <div className="flex items-center py-2 px-4">{props.headerComponents}</div>
                )}
                {props.onClose && (
                    <CloseButton
                        onClose={props.onClose}
                        className={props.closeButtonClassName}
                        iconColor={props.closeButtonIconColor}
                    />
                )}
            </div>
        </div>
        <div className="flex flex-1 overflow-auto transition">{props.children}</div>
    </div>
);

Panel.propTypes = {
    header: PropTypes.string,
    buttons: PropTypes.node,
    className: PropTypes.string,
    children: PropTypes.node.isRequired,
    onClose: PropTypes.func,
    closeButtonClassName: PropTypes.string,
    closeButtonIconColor: PropTypes.string,
    headerComponents: PropTypes.element
};

Panel.defaultProps = {
    header: ' ',
    buttons: null,
    className: 'w-full',
    onClose: null,
    closeButtonClassName: '',
    closeButtonIconColor: '',
    headerComponents: null
};

export default Panel;
