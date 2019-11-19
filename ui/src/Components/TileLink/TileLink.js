import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { useTheme } from 'Containers/ThemeProvider';
import Loader from 'Components/Loader';

export const POSITION = {
    FIRST: 'first',
    MIDDLE: 'middle',
    LAST: 'last'
};

const getClassNameByPosition = position => {
    if (position === POSITION.FIRST) {
        return 'border-r-0 rounded-r-none';
    }
    if (position === POSITION.MIDDLE) {
        return 'border-r-0 rounded-r-none rounded-l-none';
    }
    if (position === POSITION.LAST) {
        return 'rounded-l-none';
    }
    return '';
};

const TileLink = ({ text, superText, subText, icon, url, loading, isError, position }) => {
    const { isDarkMode } = useTheme();

    const className = getClassNameByPosition(position);

    const content = loading ? (
        <Loader className="text-base-100" message="" transparent />
    ) : (
        <div className="flex flex-col text-center">
            {superText && (
                <div
                    className="text-3xl tracking-widest pb-1 text-base-600"
                    data-testid="tileLinkSuperText"
                >
                    {superText}
                </div>
            )}
            <div
                className="flex items-center font-600 font-condensed text-base-600 uppercase justify-center"
                data-test-id="tile-link-value"
            >
                {text} {icon && <div className="ml-1">{icon}</div>}
            </div>
            {subText && (
                <div className="text-sm pt-1 tracking-wide font-condensed font-600">{subText}</div>
            )}
        </div>
    );
    let classes = '';
    const positionClasses = `flex flex-col items-center justify-center py-2 px-2 lg:px-4 min-w-20 lg:min-w-24 border-2 rounded min-h-14`;
    const colors = 'text-base-600 hover:bg-base-200 border-primary-400 bg-base-100';
    const darkModeColors = 'text-base-600 hover:bg-primary-200 border-primary-400';
    const errorColors = 'text-alert-700 bg-alert-200 hover:bg-alert-300 border-alert-400';
    const errorDarkModeColors = 'text-base-100 bg-alert-400 hover:bg-alert-500 border-alert-400';

    if (isError) {
        classes = `${positionClasses} ${isDarkMode ? errorDarkModeColors : errorColors}`;
    } else {
        classes = `${positionClasses} ${isDarkMode ? darkModeColors : colors}`;
    }
    classes += ` ${className}`;
    return (
        <Link to={url} className="no-underline" data-test-id="tile-link">
            <div className={classes}>{content}</div>
        </Link>
    );
};

TileLink.propTypes = {
    text: PropTypes.string.isRequired,
    superText: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
    subText: PropTypes.string,
    icon: PropTypes.element,
    url: PropTypes.string.isRequired,
    loading: PropTypes.bool,
    isError: PropTypes.bool,
    position: PropTypes.oneOf(Object.values(POSITION))
};

TileLink.defaultProps = {
    isError: false,
    position: null,
    loading: false,
    superText: null,
    subText: null,
    icon: null
};

export default TileLink;
