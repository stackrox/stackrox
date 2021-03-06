import React from 'react';
import PropTypes from 'prop-types';

import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';

const RowActionButton = ({ text, icon, border, className, onClick, dataTestId, disabled }) => (
    <Tooltip content={<TooltipOverlay>{text}</TooltipOverlay>}>
        <button
            type="button"
            className={`p-1 px-4 ${className} ${border}`}
            onClick={onClick}
            data-testid={dataTestId}
            disabled={disabled}
        >
            {icon}
        </button>
    </Tooltip>
);

RowActionButton.propTypes = {
    text: PropTypes.string.isRequired,
    icon: PropTypes.element.isRequired,
    border: PropTypes.string,
    className: PropTypes.string,
    onClick: PropTypes.func.isRequired,
    dataTestId: PropTypes.string,
    disabled: PropTypes.bool,
};

RowActionButton.defaultProps = {
    className: 'hover:bg-primary-200 text-primary-600 hover:text-primary-700',
    border: '',
    dataTestId: '',
    disabled: false,
};

export default RowActionButton;
