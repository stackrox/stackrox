import React from 'react';
import PropTypes from 'prop-types';

export const defaultHeaderClassName = 'flex w-full h-12 word-break';

const Widget = props => (
    <div
        className={`flex flex-col h-full border border-base-400 ${props.className}`}
        data-test-id="widget"
    >
        <div className="border-b border-base-400">
            <div className={props.headerClassName}>
                <div
                    className="flex flex-1 text-base-600 uppercase items-center tracking-wide pl-4 pt-1 leading-normal font-700"
                    data-test-id="widget-header"
                >
                    {props.header}
                </div>
                {props.headerComponents && (
                    <div className="flex items-center pr-3 relative">{props.headerComponents}</div>
                )}
            </div>
        </div>
        <div className={`flex h-full overflow-y-auto ${props.bodyClassName}`}>{props.children}</div>
    </div>
);

Widget.propTypes = {
    header: PropTypes.string,
    headerClassName: PropTypes.string,
    bodyClassName: PropTypes.string,
    className: PropTypes.string,
    children: PropTypes.node.isRequired,
    headerComponents: PropTypes.element
};

Widget.defaultProps = {
    header: '',
    headerClassName: defaultHeaderClassName,
    bodyClassName: null,
    className: 'w-full',
    headerComponents: null
};

export default Widget;
