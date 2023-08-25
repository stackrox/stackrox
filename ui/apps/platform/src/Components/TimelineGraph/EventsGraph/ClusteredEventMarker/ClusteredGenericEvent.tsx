import React, { forwardRef } from 'react';

import { getNumEventsBackgroundWidth, getNumEventsText } from './clusteredEventMarkerUtils';

type ClusteredGenericEventProps = {
    size: number;
    numEvents: number;
};

const ClusteredGenericEvent = forwardRef<SVGSVGElement, ClusteredGenericEventProps>(
    ({ size, numEvents }, ref) => {
        const numEventsBackgroundWidth = getNumEventsBackgroundWidth(numEvents);
        const numEventsText = getNumEventsText(numEvents);
        return (
            <svg
                className="cursor-pointer"
                data-testid="clustered-generic-event"
                width={size}
                height={size}
                viewBox="0 0 23 22"
                xmlns="http://www.w3.org/2000/svg"
                ref={ref}
            >
                <g fill="none" fillRule="evenodd">
                    <rect fill="#4FAFD3" y="7" width="14.5" height="14.5" rx="7.25" />
                    <g transform="translate(9)">
                        <rect
                            stroke="#8FCFE7"
                            fill="#E2F7FF"
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
                            fill="#2A7895"
                        >
                            <tspan x="2.521" y="10">
                                {numEventsText}
                            </tspan>
                        </text>
                    </g>
                </g>
            </svg>
        );
    }
);

ClusteredGenericEvent.displayName = 'ClusteredGenericEvent';

export default ClusteredGenericEvent;
