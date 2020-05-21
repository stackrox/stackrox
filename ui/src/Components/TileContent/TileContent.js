import React from 'react';
import PropTypes from 'prop-types';

const TileContent = ({
    dataTestId,
    className,
    superText,
    subText,
    icon,
    text,
    short,
    textColorClass,
    textWrap,
}) => {
    return (
        <div
            className={`flex flex-col text-center justify-around ${textColorClass} ${className}`}
            data-testid={dataTestId}
        >
            {superText !== null && (
                <div className="text-2xl tracking-widest pb-1" data-testid="tileLinkSuperText">
                    {superText}
                </div>
            )}
            {icon !== null && <div className="p-1 flex justify-center">{icon}</div>}
            <div
                className={`flex ${
                    !textWrap && 'whitespace-no-wrap'
                } items-center font-600 font-condensed uppercase justify-center text-base`}
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
    dataTestId: PropTypes.string,
    className: PropTypes.string,
    superText: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
    subText: PropTypes.string,
    icon: PropTypes.element,
    text: PropTypes.string.isRequired,
    short: PropTypes.bool,
    textColorClass: PropTypes.string,
    textWrap: PropTypes.bool,
};

TileContent.defaultProps = {
    dataTestId: 'tile-content',
    className: '',
    superText: null,
    subText: null,
    icon: null,
    short: false,
    textColorClass: 'text-base-600',
    textWrap: false,
};

export default TileContent;
