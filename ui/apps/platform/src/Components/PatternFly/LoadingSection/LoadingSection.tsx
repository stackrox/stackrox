import React from 'react';
import { ClipLoader } from 'react-spinners';
import { Flex } from '@patternfly/react-core';

interface LoadingSectionProps {
    message?: string;
}

const LoadingSection = ({ message = 'Loading...' }: LoadingSectionProps) => (
    <Flex className="pf-u-flex-direction-column pf-u-h-100 pf-u-justify-content-center pf-u-align-items-center">
        <ClipLoader color="white" loading size={20} />
        <div className="pf-u-mt-sm pf-u-color-light-100">{message}</div>
    </Flex>
);

export default LoadingSection;
