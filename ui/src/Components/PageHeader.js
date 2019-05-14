import React from 'react';
import PropTypes from 'prop-types';

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

export default PageHeader;
