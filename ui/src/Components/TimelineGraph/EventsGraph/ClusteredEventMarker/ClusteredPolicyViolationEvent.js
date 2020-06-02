/* eslint-disable react/display-name */
import React, { forwardRef } from 'react';

import { clusteredEventPropTypes } from 'constants/propTypes/timelinePropTypes';
import { getNumEventsBackgroundWidth, getNumEventsText } from './clusteredEventMarkerUtils';

const ClusteredPolicyViolationEvent = forwardRef(({ size, numEvents }, ref) => {
    const numEventsBackgroundWidth = getNumEventsBackgroundWidth(numEvents);
    const numEventsText = getNumEventsText(numEvents);
    return (
        <svg
            data-testid="clustered-policy-violation-event"
            width={size}
            height={size}
            viewBox="0 0 23 22"
            version="1.1"
            xmlns="http://www.w3.org/2000/svg"
            ref={ref}
        >
            <g fill="none" fillRule="evenodd">
                <g transform="translate(0 6.1)">
                    <rect fill="#FF5782" width="14.5" height="14.5" rx="2.43" />
                    <path
                        d="M8.071 8.37H6.616l-.325-5.584h2.12L8.07 8.37zm-1.833 2.22c0-.346.091-.614.275-.804.184-.19.458-.286.82-.286.364 0 .639.094.825.282.186.189.279.458.279.808 0 .345-.096.613-.29.804-.192.19-.463.285-.813.285-.359 0-.631-.096-.817-.289-.186-.192-.28-.46-.28-.8z"
                        fill="#FFF"
                        fillRule="nonzero"
                    />
                </g>
                <g transform="translate(9)">
                    <rect
                        stroke="#D48D9F"
                        fill="#FFEBF0"
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
                        fill="#9D2E4B"
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

ClusteredPolicyViolationEvent.propTypes = clusteredEventPropTypes;

export default ClusteredPolicyViolationEvent;
