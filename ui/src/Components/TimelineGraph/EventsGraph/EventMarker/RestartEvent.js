/* eslint-disable react/display-name */
import React, { forwardRef } from 'react';
import PropTypes from 'prop-types';

const RestartEvent = forwardRef(({ size }, ref) => {
    return (
        <svg
            data-testid="restart-event"
            width={size}
            height={size}
            viewBox="0 0 16 15"
            xmlns="http://www.w3.org/2000/svg"
            ref={ref}
            fillRule="evenodd"
            clipRule="evenodd"
            strokeLinejoin="round"
            strokeMiterlimit="2"
        >
            <path
                d="M9.202 1.183l6.612 11.268a1.354 1.354 0 01-.508 1.866c-.211.12-.451.183-.694.183H1.388A1.377 1.377 0 010 13.134c0-.24.064-.475.186-.683L6.798 1.183a1.4 1.4 0 012.404 0z"
                fill="#ff9064"
            />
        </svg>
    );
});

RestartEvent.propTypes = {
    size: PropTypes.number.isRequired,
};

export default RestartEvent;
