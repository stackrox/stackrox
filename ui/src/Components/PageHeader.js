import React from 'react';
import PropTypes from 'prop-types';
import { TooltipDiv } from './Panel';

const renderSubHeader = subHeader => {
    if (!subHeader) return null;
    return <div className="mt-1 italic capitalize opacity-75">{subHeader}</div>;
};

const PageHeader = props => (
    <div
        className={`flex h-18 px-4 bg-base-100 border-b border-base-400 w-full flex-no-shrink ${
            props.classes
        }`}
        style={props.bgStyle}
    >
        <div className="min-w-max pr-4 self-center">
            <div className="uppercase text-lg tracking-widest font-700 pt-1">{props.header}</div>
            {renderSubHeader(props.subHeader)}
        </div>
        <div className="flex w-full items-center">{props.children}</div>
    </div>
);

PageHeader.propTypes = {
    header: PropTypes.string.isRequired,
    subHeader: PropTypes.string,
    classes: PropTypes.string,
    bgStyle: PropTypes.shape({}),
    children: PropTypes.oneOfType([PropTypes.element, PropTypes.arrayOf(PropTypes.element)])
};

PageHeader.defaultProps = {
    children: null,
    subHeader: null,
    classes: null,
    bgStyle: null
};

const maxSize = 1000;

export const PageHeaderComponent = props => {
    let headerText = '';
    if (props.selectionCount !== 0) {
        headerText = `${props.selectionCount} ${props.type}${
            props.selectionCount === 1 ? '' : 's'
        } Selected`;
    } else {
        headerText = `${props.length}${props.length === maxSize ? '+' : ''} ${props.type}${
            props.length === 1 ? '' : 's'
        } ${props.isViewFiltered ? 'Matched' : ''} ${
            props.length === maxSize ? 'are available' : ''
        }
    `;
    }
    let component = <TooltipDiv header={headerText} isUpperCase />;
    if (props.length === maxSize) {
        component = (
            <div className="pt-2">
                {component}
                <div className="pl-4 opacity-75 italic">
                    Please add a filter to narrow down your results.
                </div>
            </div>
        );
    }
    return component;
};

PageHeaderComponent.propTypes = {
    length: PropTypes.number.isRequired,
    selectionCount: PropTypes.number,
    type: PropTypes.string.isRequired,
    isViewFiltered: PropTypes.boolean
};

PageHeaderComponent.defaultProps = {
    isViewFiltered: false,
    selectionCount: 0
};

export default PageHeader;
