import React from 'react';
import PropTypes from 'prop-types';

const renderSubHeader = subHeader => {
    if (!subHeader) return null;
    return <div className="text-primary-400 mt-2 font-400 italic">{subHeader}</div>;
};

const PageHeader = props => (
    <div className="flex flex-row bg-white py-3 px-4 border-b border-primary-300 h-16 w-full">
        <div className="w-48 self-center">
            <div className="text-base-600 uppercase text-lg tracking-wide">{props.header}</div>
            {renderSubHeader(props.subHeader)}
        </div>
        <div className="flex flex-row w-full">{props.children}</div>
    </div>
);

PageHeader.propTypes = {
    header: PropTypes.string.isRequired,
    subHeader: PropTypes.string,
    children: PropTypes.oneOfType([PropTypes.element, PropTypes.arrayOf(PropTypes.element)])
};

PageHeader.defaultProps = {
    children: null,
    subHeader: null
};

export default PageHeader;
