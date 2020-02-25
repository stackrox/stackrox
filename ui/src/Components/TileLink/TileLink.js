import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

import { useTheme } from 'Containers/ThemeProvider';
import Loader from 'Components/Loader';
import TileContent from 'Components/TileContent';

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

const TileLink = ({
    text,
    superText,
    subText,
    icon,
    url,
    loading,
    isError,
    position,
    short,
    dataTestId
}) => {
    const { isDarkMode } = useTheme();

    const positionClassName = getClassNameByPosition(position);

    const content = loading ? (
        <Loader className="text-base-100" message="" transparent />
    ) : (
        <TileContent
            superText={superText}
            text={text}
            icon={icon}
            subText={subText}
            short={short}
        />
    );
    let classes = '';
    const positionClasses = `flex flex-col items-center justify-center py-2 border-2 rounded min-w-20 px-2 lg:px-4  ${
        short ? '' : 'lg:min-w-24'
    }`;
    const colors = 'text-base-600 hover:bg-base-200 border-primary-400 bg-base-100';
    const darkModeColors = 'text-base-600 hover:bg-primary-200 border-primary-400';
    const errorColors = 'text-alert-700 bg-alert-200 hover:bg-alert-300 border-alert-400';
    const errorDarkModeColors = 'text-base-100 bg-alert-400 hover:bg-alert-500 border-alert-400';

    if (isError) {
        classes = `${positionClasses} ${isDarkMode ? errorDarkModeColors : errorColors}`;
    } else {
        classes = `${positionClasses} ${isDarkMode ? darkModeColors : colors}`;
    }
    classes += ` ${positionClassName} ${short ? 'h-full' : 'min-h-14'}`;
    return (
        <Link to={url} className="no-underline" data-test-id={dataTestId}>
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
    position: PropTypes.oneOf(Object.values(POSITION)),
    short: PropTypes.bool,
    dataTestId: PropTypes.string
};

TileLink.defaultProps = {
    isError: false,
    position: null,
    loading: false,
    superText: null,
    subText: null,
    icon: null,
    short: false,
    dataTestId: 'tile-link'
};

export default TileLink;
