import React from 'react';
import PropTypes from 'prop-types';
import { AngleDownIcon, AngleUpIcon } from '@patternfly/react-icons';
import { Button, Flex } from '@patternfly/react-core';

import IconWithCount from 'Components/IconWithCount';

const CollapsibleCountsButton = ({ isOpen, onClick, children }) => {
    return (
        <Button variant="plain" className="pf-u-background-color-200" onClick={onClick}>
            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                {children}
                {isOpen && <AngleDownIcon />}
                {!isOpen && <AngleUpIcon />}
            </Flex>
        </Button>
    );
};

CollapsibleCountsButton.propTypes = {
    isOpen: PropTypes.bool,
    onClick: PropTypes.func.isRequired,
    children: PropTypes.oneOf([
        PropTypes.arrayOf(PropTypes.oneOfType(IconWithCount)),
        PropTypes.oneOfType(IconWithCount),
    ]),
};

CollapsibleCountsButton.defaultProps = {
    isOpen: false,
    children: null,
};

export default CollapsibleCountsButton;
