import React from 'react';
import PropTypes from 'prop-types';
import { ChevronDown, ChevronUp } from 'react-feather';

import Button from 'Components/Button';
import IconWithCount from 'Components/IconWithCount';

const CollapsibleCountsButton = ({ isOpen, onClick, children }) => {
    return (
        <Button
            className={`hover:bg-base-200 ${isOpen && 'bg-base-200'} p-2 border border-base-300`}
            icon={
                <>
                    {children}
                    {isOpen && <ChevronDown className="h-4 w-4 text-base-600" />}
                    {!isOpen && <ChevronUp className="h-4 w-4 text-base-600" />}
                </>
            }
            onClick={onClick}
        />
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
