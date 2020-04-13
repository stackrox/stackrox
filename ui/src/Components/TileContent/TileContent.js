import React from 'react';
import PropTypes from 'prop-types';

const TileContent = ({
    className,
    superText,
    subText,
    icon,
    text,
    short,
    textColorClass,
    textWrap
}) => {
    return (
        <div className={`flex flex-col text-center justify-around ${textColorClass} ${className}`}>
            {superText !== null && (
                <div className="text-3xl tracking-widest pb-1" data-testid="tileLinkSuperText">
                    {superText}
                </div>
            )}
            {icon !== null && <div className="p-1 flex justify-center">{icon}</div>}
            <div
                className={`flex ${!textWrap &&
                    'whitespace-no-wrap'} items-center font-600 font-condensed uppercase justify-center`}
                data-testid="tile-link-value"
            >
                {text}
            </div>
            {subText !== null && (
                <div
                    className={`${
                        short ? 'text-xs' : 'text-sm pt-1'
                    } tracking-wide font-condensed font-600`}
                >
                    {subText}
                </div>
            )}
        </div>
    );
};

TileContent.propTypes = {
    className: PropTypes.string,
    superText: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
    subText: PropTypes.string,
    icon: PropTypes.element,
    text: PropTypes.string.isRequired,
    short: PropTypes.bool,
    textColorClass: PropTypes.string,
    textWrap: PropTypes.bool
};

TileContent.defaultProps = {
    className: '',
    superText: null,
    subText: null,
    icon: null,
    short: false,
    textColorClass: 'text-base-600',
    textWrap: false
};

export default TileContent;
