import React, { useEffect, useRef, useState } from 'react';
import PropTypes from 'prop-types';
import { Tooltip } from 'react-tippy';
import { throttle } from 'lodash';
import 'rc-tooltip/assets/bootstrap.css';
import CloseButton from './CloseButton';

export const headerClassName = 'flex w-full h-12';

const TooltipDiv = ({ header, isUpperCase }) => {
    const titleClassName = isUpperCase ? 'uppercase' : 'capitalize';
    const tooltipContent = <span className="text-sm">{header}</span>;

    const parentRef = useRef(null);
    const tooltipRef = useRef(null);
    const [allowTooltip, setAllowTooltip] = useState(false);
    let content = (
        <div ref={tooltipRef} className="flex-none">
            {header}
        </div>
    );
    const tooltipFn = () => {
        setAllowTooltip(false);
        if (
            parentRef.current &&
            tooltipRef.current &&
            parentRef.current.offsetWidth <= tooltipRef.current.offsetWidth
        ) {
            setAllowTooltip(true);
        }
    };

    function setWindowResize() {
        window.addEventListener('resize', throttle(tooltipFn, 100));

        const cleanup = () => {
            window.removeEventListener('resize');
        };

        return cleanup;
    }

    if (allowTooltip) {
        content = (
            <Tooltip
                useContext
                position="top"
                trigger="mouseenter"
                arrow
                html={tooltipContent}
                className="truncate"
                unmountHTMLWhenHide
            >
                <div ref={tooltipRef} className="truncate flex-none">
                    {header}
                </div>
            </Tooltip>
        );
    }
    useEffect(tooltipFn, [header]);
    useEffect(setWindowResize, []);
    return (
        <div
            ref={parentRef}
            className={`overflow-hidden mx-4 flex text-base-600 items-center tracking-wide leading-normal font-700 ${titleClassName}`}
            data-test-id="panel-header"
        >
            {content}
        </div>
    );
};

TooltipDiv.propTypes = {
    header: PropTypes.string,
    isUpperCase: PropTypes.bool
};

TooltipDiv.defaultProps = {
    header: ' ',
    isUpperCase: true
};

const Panel = props => (
    <div
        className={`flex flex-col h-full border-r border-base-400 ${props.className}`}
        data-test-id="panel"
    >
        <div className="border-b border-base-400 flex-no-wrap">
            <div className={props.headerClassName}>
                {props.leftButtons && (
                    <div className="flex items-center pr-3 relative border-base-400 border-r hover:bg-primary-300 hover:border-primary-300">
                        {props.leftButtons}
                    </div>
                )}
                {props.headerTextComponent ? (
                    props.headerTextComponent
                ) : (
                    <TooltipDiv header={props.header} isUpperCase={props.isUpperCase} />
                )}

                <div className="flex flex-1 items-center justify-end pl-3 relative">
                    {props.headerComponents && props.headerComponents}
                    {props.onClose && (
                        <CloseButton
                            onClose={props.onClose}
                            className={props.closeButtonClassName}
                            iconColor={props.closeButtonIconColor}
                        />
                    )}
                </div>
            </div>
        </div>
        <div className={`flex h-full overflow-y-auto ${props.bodyClassName}`}>{props.children}</div>
    </div>
);

Panel.propTypes = {
    header: PropTypes.string,
    headerTextComponent: PropTypes.element,
    headerClassName: PropTypes.string,
    bodyClassName: PropTypes.string,
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
    className: 'w-full',
    onClose: null,
    closeButtonClassName: '',
    closeButtonIconColor: '',
    headerComponents: null,
    leftButtons: null,
    isUpperCase: true
};

export default Panel;
