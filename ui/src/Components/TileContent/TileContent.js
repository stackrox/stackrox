import React from 'react';
import PropTypes from 'prop-types';

const TileContent = ({ className, superText, subText, icon, text, short, textColorClass }) => {
    return (
        <div className={`flex flex-col text-center justify-around ${textColorClass} ${className}`}>
            {superText && (
                <div className="text-3xl tracking-widest pb-1" data-test-id="tileLinkSuperText">
                    {superText}
                </div>
            )}
            {icon && <div className="p-1 flex justify-center">{icon}</div>}
            <div
                className="flex items-center font-600 font-condensed uppercase justify-center"
                data-test-id="tile-link-value"
            >
                {text}
            </div>
            {subText && (
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
    textColorClass: PropTypes.string
};

TileContent.defaultProps = {
    className: '',
    superText: null,
    subText: null,
    icon: null,
    short: false,
    textColorClass: 'text-base-600'
};

export default TileContent;
