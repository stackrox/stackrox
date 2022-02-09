import React from 'react';
import PropTypes from 'prop-types';
import { Spinner, Bullseye, Flex } from '@patternfly/react-core';

function IconWithCount({ Icon, count, isLoading }) {
    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
            {isLoading ? (
                <Bullseye>
                    <Spinner isSVG size="lg" />
                </Bullseye>
            ) : (
                <>
                    <span className="pf-u-mr-sm">{count}</span>
                    <Icon />
                </>
            )}
        </Flex>
    );
}

IconWithCount.propTypes = {
    Icon: PropTypes.elementType.isRequired,
    count: PropTypes.number.isRequired,
    isLoading: PropTypes.bool,
};

IconWithCount.defaultProps = {
    isLoading: false,
};

export default IconWithCount;
