import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { useTheme } from 'Containers/ThemeProvider';
import Loader from 'Components/Loader';

const TileLink = ({ value, caption, to, loading }) => {
    const { isDarkMode } = useTheme();

    const content = loading ? (
        <Loader className="text-base-100" message="" transparent />
    ) : (
        <>
            <div className="text-3xl tracking-widest" data-test-id="tile-link-value">
                {value}
            </div>
            <div className="text-sm pt-1 tracking-wide uppercase font-condensed">
                {value === 1 ? caption : `${caption}s`}
            </div>
        </>
    );
    const classes = `flex flex-col items-center justify-center px-2 lg:px-4 min-w-20 lg:min-w-24 border-2 border-primary-400 ${
        isDarkMode ? 'text-base-600 hover:bg-primary-200' : 'text-base-100 hover:bg-primary-800'
    } rounded min-h-14 uppercase`;
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
    loading: PropTypes.bool.isRequired
};

export default TileLink;
