import React from 'react';
import { Flex, Spinner } from '@patternfly/react-core';

import './LoadingSection.css';

interface LoadingSectionProps {
    /** The message to display below the Spinner */
    message?: string;
    /** The size of the Spinner to pass through to the PatternFly component */
    spinnerSize?: 'sm' | 'md' | 'lg' | 'xl';
    /** Should the color of the Spinner and text be inverted from the default theme? (Defaults to `false`, which results in a white color.) */
    isColorInverted?: boolean;
}

const LoadingSection = ({
    message = 'Loading...',
    spinnerSize = 'lg',
    isColorInverted: invertColors = false,
}: LoadingSectionProps) => (
    <Flex
        className={`loading-section ${
            invertColors ? 'loading-section-inverted' : ''
        } pf-u-flex-direction-column pf-u-h-100 pf-u-justify-content-center pf-u-align-items-center`}
    >
        <Spinner aria-valuetext={message} size={spinnerSize} />
        <div className="pf-u-mt-sm">{message}</div>
    </Flex>
);

export default LoadingSection;
