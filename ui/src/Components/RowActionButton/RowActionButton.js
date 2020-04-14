import React from 'react';
import PropTypes from 'prop-types';

import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';

const RowActionButton = ({ text, icon, border, className, onClick, dataTestId }) => (
    <Tooltip content={<TooltipOverlay>{text}</TooltipOverlay>}>
        <button
            type="button"
            className={`p-1 px-4 ${className} ${border}`}
            onClick={onClick}
            data-testid={dataTestId}
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
    dataTestId: PropTypes.string
};

RowActionButton.defaultProps = {
    className: 'hover:bg-primary-200 text-primary-600 hover:text-primary-700',
    border: '',
    dataTestId: ''
};

export default RowActionButton;
