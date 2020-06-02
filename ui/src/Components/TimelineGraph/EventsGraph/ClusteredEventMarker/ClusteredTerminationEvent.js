/* eslint-disable react/display-name */
import React, { forwardRef } from 'react';

import { clusteredEventPropTypes } from 'constants/propTypes/timelinePropTypes';
import { getNumEventsBackgroundWidth, getNumEventsText } from './clusteredEventMarkerUtils';

const ClusteredTerminationEvent = forwardRef(({ size, numEvents }, ref) => {
    const numEventsBackgroundWidth = getNumEventsBackgroundWidth(numEvents);
    const numEventsText = getNumEventsText(numEvents);
    return (
        <svg
            data-testid="clustered-termination-event"
            width={size}
            height={size}
            viewBox="0 0 22 22"
            xmlns="http://www.w3.org/2000/svg"
            ref={ref}
        >
            <g fill="none" fillRule="evenodd">
                <path
                    d="M9.53 20.792l6.849-11.67a1.402 1.402 0 00-.527-1.932 1.456 1.456 0 00-.718-.19H1.438C.644 7 0 7.633 0 8.415c0 .248.066.492.193.707l6.848 11.67a1.45 1.45 0 002.49 0z"
                    fill="#FF9064"
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
                        fontFamily="OpenSans-Bold, Open Sans"
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
            </g>
        </svg>
    );
});

ClusteredTerminationEvent.propTypes = clusteredEventPropTypes;

export default ClusteredTerminationEvent;
