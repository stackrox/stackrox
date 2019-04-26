import React from 'react';
import PropTypes from 'prop-types';
import 'rc-tooltip/assets/bootstrap.css';
import CloseButton from './CloseButton';

export const headerClassName = 'flex w-full word-break';

const Panel = props => {
    const titleClassName = props.isUpperCase ? 'uppercase' : 'capitalize';
    const headerText = (
        <div
            className={`m-1 flex flex-1 text-base-600 items-center tracking-wide leading-normal font-700 lg:ml-2 lg:mr-2 ${titleClassName}`}
            data-test-id="panel-header"
        >
            {props.header}
        </div>
    );
    return (
        <div
            className={`flex flex-col h-full border-r border-base-400 min-w-0 ${props.className}`}
            data-test-id="panel"
        >
            <div className="border-b border-base-400">
                <div className={props.headerClassName}>
                    {props.leftButtons && (
                        <div className="flex items-center pr-3 relative border-base-400 border-r hover:bg-primary-300 hover:border-primary-300">
                            {props.leftButtons}
                        </div>
                    )}

                    <div className="mr-2 ml-2 mb-1 lg:ml-0 lg:mr-0 lg:mb-0 lg:flex pt-1 justify-center flex-grow">
                        {props.headerTextComponent ? props.headerTextComponent : headerText}
                        <div className="panel-actions relative flex items-center">
                            {props.buttons}
                        </div>
                    </div>

                    {props.headerComponents && (
                        <div className="flex items-center pr-3 relative">
                            {props.headerComponents}
                        </div>
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
            <div className={`flex h-full overflow-y-auto ${props.bodyClassName}`}>
                {props.children}
            </div>
        </div>
    );
};

Panel.propTypes = {
    header: PropTypes.string,
    headerTextComponent: PropTypes.element,
    headerClassName: PropTypes.string,
    bodyClassName: PropTypes.string,
    buttons: PropTypes.node,
    className: PropTypes.string,
    children: PropTypes.node.isRequired,
    onClose: PropTypes.func,
    closeButtonClassName: PropTypes.string,
    closeButtonIconColor: PropTypes.string,
    headerComponents: PropTypes.element,
    leftButtons: PropTypes.node,
    isUpperCase: PropTypes.bool
};

Panel.defaultProps = {
    header: ' ',
    headerTextComponent: null,
    headerClassName,
    bodyClassName: null,
    buttons: null,
    className: 'w-full',
    onClose: null,
    closeButtonClassName: '',
    closeButtonIconColor: '',
    headerComponents: null,
    leftButtons: null,
    isUpperCase: true
};

export default Panel;
