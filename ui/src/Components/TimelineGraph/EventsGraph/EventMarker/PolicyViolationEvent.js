/* eslint-disable react/display-name */
import React, { forwardRef } from 'react';
import PropTypes from 'prop-types';

const PolicyViolationEvent = forwardRef(({ size }, ref) => {
    return (
        <svg
            data-testid="policy-violation-event"
            width={size}
            height={size}
            viewBox="0 0 15 15"
            version="1.1"
            xmlns="http://www.w3.org/2000/svg"
            ref={ref}
        >
            <g transform="translate(0 .1)" fill="none">
                <rect fill="#FF5782" width="14.58" height="14.58" rx="2.43" />
                <path
                    d="M8.071 8.37H6.616l-.325-5.584h2.12L8.07 8.37zm-1.833 2.22c0-.346.091-.614.275-.804.184-.19.458-.286.82-.286.364 0 .639.094.825.282.186.189.279.458.279.808 0 .345-.096.613-.29.804-.192.19-.463.285-.813.285-.359 0-.631-.096-.817-.289-.186-.192-.28-.46-.28-.8z"
                    fill="#FFF"
                />
            </g>
        </svg>
    );
});

PolicyViolationEvent.propTypes = {
    size: PropTypes.number.isRequired,
};

export default PolicyViolationEvent;
