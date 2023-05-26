import React from 'react';
import PropTypes from 'prop-types';

import { useTheme } from 'Containers/ThemeProvider';
import SubHeader from 'Components/SubHeader';

const PageHeader = ({ header, subHeader, classes, bgStyle, children }) => {
    const { isDarkMode } = useTheme();

    return (
        <div
            className={`flex h-18 px-4 w-full flex-shrink-0 z-10 border-b border-base-400 ${classes} ${
                !isDarkMode ? 'bg-base-100' : 'bg-base-0'
            }`}
            style={bgStyle}
            data-testid="page-header"
        >
            <div className="min-w-max pr-4 self-center">
                <h1 data-testid="header-text" className="text-lg font-700">
                    {header}
                </h1>
                {subHeader && <SubHeader text={subHeader} />}
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
    children: PropTypes.oneOfType([PropTypes.element, PropTypes.arrayOf(PropTypes.element)]),
};

PageHeader.defaultProps = {
    children: null,
    subHeader: null,
    classes: '',
    bgStyle: null,
};

export default PageHeader;
