import React from 'react';
import PropTypes from 'prop-types';
import { ClipLoader } from 'react-spinners';
import { Flex } from '@patternfly/react-core';

const LoadingSection = ({ message }) => (
    <Flex className="pf-u-flex-direction-column pf-u-h-100 pf-u-justify-content-center pf-u-align-items-center">
        <ClipLoader color="white" loading size={20} />
        <div className="pf-u-mt-sm pf-u-color-light-100">{message}</div>
    </Flex>
);

LoadingSection.propTypes = {
    message: PropTypes.string,
};

LoadingSection.defaultProps = {
    message: 'Loading...',
};

export default LoadingSection;
