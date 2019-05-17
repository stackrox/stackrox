import React from 'react';
import PropTypes from 'prop-types';

const sizeClassMap = {
    UNSET: '',
    SMALL: 'h-12',
    MEDIUM: 'h-18',
    LARGE: 'h-24'
};

const AppBanner = ({ enabled, text, color, size, backgroundColor }) => {
    if (!enabled) return null;
    return (
        <div
            className={`${sizeClassMap[size]} px-2 py-1 whitespace-pre text-center`}
            style={{ color, backgroundColor }}
        >
            {text}
        </div>
    );
};

AppBanner.propTypes = {
    enabled: PropTypes.bool,
    text: PropTypes.string,
    color: PropTypes.string,
    size: PropTypes.string,
    backgroundColor: PropTypes.string
};

AppBanner.defaultProps = {
    enabled: false,
    text: '',
    color: '',
    size: 'UNSET',
    backgroundColor: ''
};

export default AppBanner;
