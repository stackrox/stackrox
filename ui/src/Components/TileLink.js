import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { useTheme } from 'Containers/ThemeProvider';
import Loader from 'Components/Loader';
import pluralize from 'pluralize';

const TileLink = ({ value, caption, to, loading, isError, className }) => {
    const { isDarkMode } = useTheme();

    const content = loading ? (
        <Loader className="text-base-100" message="" transparent />
    ) : (
        <>
            <div className="text-3xl tracking-widest" data-test-id="tile-link-value">
                {value}
            </div>
            <div className="text-sm pt-1 tracking-wide uppercase font-condensed">
                {value === 1 ? caption : `${pluralize(caption)}`}
            </div>
        </>
    );
    let classes = '';
    const positionClasses = `flex flex-col items-center justify-center px-2 lg:px-4 min-w-20 lg:min-w-24 border-2 ${
        className.includes('rounded') ? '' : 'rounded'
    } min-h-14 uppercase`;
    const colors = 'text-base-600 hover:bg-base-300 border-primary-400 bg-base-100';
    const darkModeColors = 'text-base-600 hover:bg-primary-200 border-primary-400';
    const errorColors = 'text-alert-700 bg-alert-200 hover:bg-alert-300 border-alert-500';
    const errorDarkModeColors = 'text-base-100 bg-alert-400 hover:bg-alert-500 border-alert-600';

    if (isError) {
        classes = `${positionClasses} ${isDarkMode ? errorDarkModeColors : errorColors}`;
    } else {
        classes = `${positionClasses} ${isDarkMode ? darkModeColors : colors}`;
    }
    classes += ` ${className}`;
    return (
        <Link to={to} className="no-underline" data-test-id="tile-link">
            <div className={classes}>{content}</div>
        </Link>
    );
};

TileLink.propTypes = {
    value: PropTypes.number.isRequired,
    caption: PropTypes.string.isRequired,
    to: PropTypes.string.isRequired,
    loading: PropTypes.bool.isRequired,
    isError: PropTypes.bool,
    className: PropTypes.string
};

TileLink.defaultProps = {
    isError: false,
    className: ''
};

export default TileLink;
