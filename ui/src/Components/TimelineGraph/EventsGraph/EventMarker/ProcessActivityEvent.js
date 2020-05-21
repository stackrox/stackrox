/* eslint-disable react/display-name */
import React, { forwardRef } from 'react';
import PropTypes from 'prop-types';

const ProcessActivityEvent = forwardRef(({ whitelisted, size }, ref) => {
    return whitelisted ? (
        <svg
            data-testid="whitelisted-process-activity-event"
            width={size}
            height={size}
            viewBox="0 0 16 16"
            xmlns="http://www.w3.org/2000/svg"
            ref={ref}
        >
            <g transform="translate(0 .1)" fill="none" fillRule="evenodd">
                <rect fill="#56DDB2" width="14.58" height="14.58" rx="2.43" />
                <path
                    d="M4.459 6.768a.807.807 0 00-1.18-.072.91.91 0 00-.067 1.24l2.685 3.17a.81.81 0 001.281-.043l4.645-6.244a.909.909 0 00-.144-1.233.808.808 0 00-1.172.151L6.48 9.153l-2.02-2.385z"
                    fill="#FFF"
                    fillRule="nonzero"
                />
            </g>
        </svg>
    ) : (
        <svg
            data-testid="process-activity-event"
            width={size}
            height={size}
            viewBox="0 0 16 16"
            xmlns="http://www.w3.org/2000/svg"
            ref={ref}
        >
            <rect
                x="689"
                y="673.1"
                width="14.58"
                height="14.58"
                rx="2.43"
                transform="translate(-689 -673)"
                fill="#5677DD"
                fillRule="evenodd"
            />
        </svg>
    );
});

ProcessActivityEvent.propTypes = {
    whitelisted: PropTypes.bool,
    size: PropTypes.number.isRequired,
};

ProcessActivityEvent.defaultProps = {
    whitelisted: false,
};

export default ProcessActivityEvent;
