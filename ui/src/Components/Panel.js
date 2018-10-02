import React from 'react';
import PropTypes from 'prop-types';
import 'rc-tooltip/assets/bootstrap.css';
import CloseButton from './CloseButton';

const Panel = props => (
    <div
        className={`flex flex-col border h-full border-t-0 border-base-400 ${props.className}`}
        data-test-id="panel"
    >
        <div className="border-b border-base-400">
            <div className="flex w-full h-12 word-break">
                <div
                    className="flex flex-1 text-base-600 uppercase items-center tracking-wide px-4 pt-1 leading-normal font-700"
                    data-test-id="panel-header"
                >
                    {props.header}
                </div>
                <div className="flex items-center px-4">{props.buttons}</div>
                {props.headerComponents && (
                    <div className="flex items-center px-4 pr-3">{props.headerComponents}</div>
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
        <div className="flex flex-1 overflow-auto">{props.children}</div>
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
