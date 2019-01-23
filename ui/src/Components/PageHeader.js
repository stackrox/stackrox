import React from 'react';
import PropTypes from 'prop-types';

const renderSubHeader = subHeader => {
    if (!subHeader) return null;
    return <div className="text-base-500 mt-1 italic capitalize">{subHeader}</div>;
};

const PageHeader = props => (
    <div className="flex bg-base-100 h-18 px-4 border-b border-base-400 w-full">
        <div className="w-48 self-center">
            <div className="text-base-600 uppercase text-lg tracking-widest font-700 pt-1">
                {props.header}
            </div>
            {renderSubHeader(props.subHeader)}
        </div>
        <div className="flex w-full items-center">{props.children}</div>
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
