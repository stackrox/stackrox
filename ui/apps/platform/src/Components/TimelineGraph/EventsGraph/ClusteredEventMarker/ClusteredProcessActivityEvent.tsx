import React, { forwardRef } from 'react';

import { getNumEventsBackgroundWidth, getNumEventsText } from './clusteredEventMarkerUtils';

type ClusteredProcessActivityEventProps = {
    inBaseline?: boolean;
    size: number;
    numEvents: number;
};

const ClusteredProcessActivityEvent = forwardRef<SVGSVGElement, ClusteredProcessActivityEventProps>(
    ({ inBaseline = false, size, numEvents }, ref) => {
        const numEventsBackgroundWidth = getNumEventsBackgroundWidth(numEvents);
        const numEventsText = getNumEventsText(numEvents);
        return inBaseline ? (
            <svg
                className="cursor-pointer"
                data-testid="clustered-process-in-baseline-activity-event"
                width={size}
                height={size}
                viewBox="0 0 23 22"
                xmlns="http://www.w3.org/2000/svg"
                ref={ref}
            >
                <g fill="none" fillRule="evenodd">
                    <g transform="translate(0 6)">
                        <rect fill="#56DDB2" width="14.58" height="14.58" rx="2.43" />
                        <path
                            d="M4.459 6.768a.807.807 0 00-1.18-.072.91.91 0 00-.067 1.24l2.685 3.17a.81.81 0 001.281-.043l4.645-6.244a.909.909 0 00-.144-1.233.808.808 0 00-1.172.151L6.48 9.153l-2.02-2.385z"
                            fill="#FFF"
                            fillRule="nonzero"
                        />
                    </g>
                    <g transform="translate(9)">
                        <rect
                            stroke="#8DD4BD"
                            fill="#EBFFF9"
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
                            fill="#2E9D7A"
                        >
                            <tspan x="2.043" y="10">
                                {numEventsText}
                            </tspan>
                        </text>
                    </g>
                </g>
            </svg>
        ) : (
            <svg
                className="cursor-pointer"
                data-testid="clustered-process-activity-event"
                width={size}
                height={size}
                viewBox="0 0 23 22"
                xmlns="http://www.w3.org/2000/svg"
                ref={ref}
            >
                <g fill="none" fillRule="evenodd">
                    <rect
                        width="14.5"
                        height="14.5"
                        rx="2.43"
                        transform="translate(0 6.1)"
                        fill="#5677DD"
                    />
                    <g transform="translate(9)">
                        <rect
                            stroke="#8D9FD4"
                            fill="#EBF0FF"
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
                            fill="#2E4A9D"
                        >
                            <tspan x="2.043" y="10">
                                {numEventsText}
                            </tspan>
                        </text>
                    </g>
                </g>
            </svg>
        );
    }
);

ClusteredProcessActivityEvent.displayName = 'ClusteredProcessActivityEvent';

export default ClusteredProcessActivityEvent;
