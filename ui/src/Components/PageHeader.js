import React from 'react';
import PropTypes from 'prop-types';
import { useTheme } from 'Containers/ThemeProvider';

const renderSubHeader = subHeader => {
    if (!subHeader) return null;
    return <div className="mt-1 italic capitalize opacity-75">{subHeader}</div>;
};

const PageHeader = ({ header, subHeader, classes, bgStyle, children }) => {
    const { isDarkMode } = useTheme();
    return (
        <div
            className={`flex h-18 px-4 w-full flex-shrink-0 z-10 border-b border-base-400 ${classes} ${
                !isDarkMode ? 'bg-base-100' : 'bg-base-0'
            }`}
            style={bgStyle}
            data-test-id="page-header"
        >
            <div className="min-w-max pr-4 self-center">
                <h1
                    data-test-id="header-text"
                    className="uppercase text-lg tracking-widest font-700 pt-1"
                >
                    {header}
                </h1>
                {renderSubHeader(subHeader)}
            </div>
            <div className="flex w-full items-center">{children}</div>
        </div>
    );
};

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
    classes: '',
    bgStyle: null
};

export default PageHeader;
