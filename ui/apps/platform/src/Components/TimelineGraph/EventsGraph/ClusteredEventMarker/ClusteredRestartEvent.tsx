import React, { forwardRef } from 'react';

import { getNumEventsBackgroundWidth, getNumEventsText } from './clusteredEventMarkerUtils';

type ClusteredRestartEventProps = {
    size: number;
    numEvents: number;
};

const ClusteredRestartEvent = forwardRef<SVGSVGElement, ClusteredRestartEventProps>(
    ({ size, numEvents }, ref) => {
        const numEventsBackgroundWidth = getNumEventsBackgroundWidth(numEvents);
        const numEventsText = getNumEventsText(numEvents);
        return (
            <svg
                className="cursor-pointer"
                data-testid="clustered-restart-event"
                width={size}
                height={size}
                viewBox="0 0 22 22"
                xmlns="http://www.w3.org/2000/svg"
                ref={ref}
                fillRule="evenodd"
                clipRule="evenodd"
            >
                <path
                    d="M9.531 7.708l6.848 11.67a1.402 1.402 0 01-.527 1.932c-.218.125-.466.19-.718.19H1.438A1.426 1.426 0 010 20.085c0-.248.066-.492.193-.707l6.848-11.67a1.449 1.449 0 012.49 0z"
                    fill="#ff9064"
                />
                <g transform="translate(8)">
                    <rect
                        stroke="#D87953"
                        fill="#FFEBE3"
                        x=".5"
                        y=".5"
                        width={numEventsBackgroundWidth}
                        height="12"
                        rx="2"
                    />
                    <text
                        fontFamily="var(--pf-global--FontFamily--sans-serif)"
                        fontSize="9"
                        fontWeight="bold"
                        letterSpacing="-.18"
                        fill="#8B4225"
                    >
                        <tspan x="2.043" y="10">
                            {numEventsText}
                        </tspan>
                    </text>
                </g>
            </svg>
        );
    }
);

ClusteredRestartEvent.displayName = 'ClusteredRestartEvent';

export default ClusteredRestartEvent;
