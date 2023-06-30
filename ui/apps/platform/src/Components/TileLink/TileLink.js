import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

import { useTheme } from 'Containers/ThemeProvider';
import Loader from 'Components/Loader';
import TileContent from 'Components/TileContent';

export const POSITION = {
    FIRST: 'first',
    MIDDLE: 'middle',
    LAST: 'last',
};

const getClassNameByPosition = (position) => {
    if (position === POSITION.LAST) {
        return 'mr-0';
    }
    return '';
};

const TileLink = ({
    text,
    superText,
    subText,
    icon,
    url,
    colorClasses,
    loading,
    isError,
    position,
    short,
}) => {
    const { isDarkMode } = useTheme();

    const positionClassName = getClassNameByPosition(position);

    const content = loading ? (
        <Loader className="text-base-100" message="" />
    ) : (
        <TileContent
            superText={superText}
            text={text}
            icon={icon}
            subText={subText}
            short={short}
            textWrap
        />
    );
    let classes = '';
    const positionClasses = `w-full flex flex-col items-center justify-center py-2 border-2 rounded-sm min-w-24 px-2`;
    const colors = 'text-base-600 hover:bg-base-200 border-base-400 bg-base-100';
    const darkModeColors = 'text-base-600 bg-base-100 border-base-400 hover:bg-base-200';
    const errorColors = 'text-alert-700 bg-alert-200 hover:bg-alert-300 border-alert-400';
    const errorDarkModeColors =
        'text-base-100 bg-alert-100 hover:bg-alert-200 border-alert-200 hover:bg-alert-300';

    if (isError) {
        classes = `${positionClasses} ${colorClasses} ${
            isDarkMode ? errorDarkModeColors : errorColors
        }`;
    } else {
        classes = `${positionClasses} ${colorClasses} ${isDarkMode ? darkModeColors : colors}`;
    }
    classes += ` ${positionClassName} ${colorClasses} ${short ? 'h-full' : 'min-h-14'}`;
    return (
        <Link to={url} className="no-underline mr-2 flex w-full" data-testid="tile-link">
            <div className={classes}>{content}</div>
        </Link>
    );
};

TileLink.propTypes = {
    text: PropTypes.string.isRequired,
    superText: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
    subText: PropTypes.string,
    colorClasses: PropTypes.string,
    icon: PropTypes.element,
    url: PropTypes.string.isRequired,
    loading: PropTypes.bool,
    isError: PropTypes.bool,
    position: PropTypes.oneOf(Object.values(POSITION)),
    short: PropTypes.bool,
};

TileLink.defaultProps = {
    isError: false,
    position: null,
    colorClasses: ' ',
    loading: false,
    superText: null,
    subText: null,
    icon: null,
    short: false,
};

export default TileLink;
