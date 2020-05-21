/* eslint-disable react/display-name */
import React, { forwardRef } from 'react';
import PropTypes from 'prop-types';

const TerminationEvent = forwardRef(({ size }, ref) => {
    return (
        <svg
            data-testid="termination-event"
            width={size}
            height={size}
            viewBox="0 0 16 16"
            xmlns="http://www.w3.org/2000/svg"
            ref={ref}
        >
            <path
                d="M9.202 13.817l6.612-11.268a1.354 1.354 0 00-.508-1.866A1.406 1.406 0 0014.612.5H1.388C.621.5 0 1.112 0 1.866c0 .24.064.475.186.683l6.612 11.268a1.4 1.4 0 002.404 0z"
                fill="#FF9064"
                fillRule="evenodd"
            />
        </svg>
    );
});

TerminationEvent.propTypes = {
    size: PropTypes.number.isRequired,
};

export default TerminationEvent;
