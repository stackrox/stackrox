import React from 'react';
import { Flex, Spinner } from '@patternfly/react-core';

import './LoadingSection.css';

interface LoadingSectionProps {
    /** The message to display below the Spinner */
    message?: string;
    /** The size of the Spinner to pass through to the PatternFly component */
    spinnerSize?: 'sm' | 'md' | 'lg' | 'xl';
    /** Should the color of the Spinner and text be light or dark? (Defaults to 'light')
     * Note that 'light' means that the text and spinner will be light in color.
     */
    variant?: 'light' | 'dark';
}

const LoadingSection = ({
    message = 'Loading...',
    spinnerSize = 'lg',
    variant = 'light',
}: LoadingSectionProps) => (
    <Flex
        className={`loading-section ${
            variant === 'light' ? 'pf-m-light' : 'pf-m-dark'
        } pf-u-flex-direction-column pf-u-h-100 pf-u-justify-content-center pf-u-align-items-center`}
    >
        <Spinner aria-valuetext={message} size={spinnerSize} />
        <div className="pf-u-mt-sm">{message}</div>
    </Flex>
);

export default LoadingSection;
