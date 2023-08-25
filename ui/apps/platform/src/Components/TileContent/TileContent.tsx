import React, { ReactElement } from 'react';

type TileContentProps = {
    dataTestId?: string;
    className?: string;
    superText?: string | number;
    subText?: string;
    icon?: ReactElement | null;
    text: string;
    short?: boolean;
    textColorClass?: string;
    textWrap?: boolean;
};

const TileContent = ({
    dataTestId = 'tile-content',
    className = '',
    superText = '',
    subText = '',
    icon = null,
    text,
    short = false,
    textColorClass = 'text-base-600',
    textWrap = false,
}: TileContentProps): ReactElement => {
    return (
        <div
            className={`flex flex-col text-center justify-around ${textColorClass} ${className}`}
            data-testid={dataTestId}
        >
            {superText !== '' && (
                <div className="text-lg pb-1" data-testid="tileLinkSuperText">
                    {superText}
                </div>
            )}
            {icon && <div className="p-1 flex justify-center">{icon}</div>}
            <div
                className={`flex ${
                    !textWrap ? 'whitespace-nowrap' : ''
                } items-center justify-center text-base`}
                data-testid="tile-link-value"
            >
                {text}
            </div>
            {subText && <div className={`${short ? 'text-xs' : 'text-sm pt-1'}`}>{subText}</div>}
        </div>
    );
};

export default TileContent;
