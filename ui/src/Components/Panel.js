import React from 'react';
import PropTypes from 'prop-types';
import 'rc-tooltip/assets/bootstrap.css';
import CloseButton from './CloseButton';

export const headerClassName = 'flex w-full h-12 word-break';

const Panel = props => (
    <div
        className={`flex flex-col h-full border border-base-400 ${props.className}`}
        data-test-id="panel"
    >
        <div className="border-b border-base-400">
            <div className={props.headerClassName}>
                <div
                    className="flex flex-1 text-base-600 uppercase items-center tracking-wide pl-4 pt-1 leading-normal font-700"
                    data-test-id="panel-header"
                >
                    {props.header}
                </div>
                <div className="panel-actions relative flex items-center mr-2">{props.buttons}</div>
                {props.headerComponents && (
                    <div className="flex items-center pr-3 relative">{props.headerComponents}</div>
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
        <div className={`flex h-full overflow-y-auto ${props.bodyClassName}`}>{props.children}</div>
    </div>
);

Panel.propTypes = {
    header: PropTypes.string,
    headerClassName: PropTypes.string,
    bodyClassName: PropTypes.string,
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
    headerClassName,
    bodyClassName: null,
    buttons: null,
    className: 'w-full',
    onClose: null,
    closeButtonClassName: '',
    closeButtonIconColor: '',
    headerComponents: null
};

export default Panel;
