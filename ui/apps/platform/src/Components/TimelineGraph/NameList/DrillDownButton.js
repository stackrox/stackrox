import React from 'react';
import PropTypes from 'prop-types';
import { ChevronRight } from 'react-feather';
import { Tooltip } from '@patternfly/react-core';

import Button from 'Components/Button';

const DrillDownButton = ({ tooltip, onClick }) => {
    let drillDownButton = null;

    // used to position the button in the center right of it's parent div
    const positionClassName = 'absolute center-y right-0 transform translate-x-1/2';

    drillDownButton = (
        <Button
            dataTestId="timeline-drill-down-button"
            className={`${
                !tooltip && positionClassName
            } bg-base-100 border border-primary-300 py-1 rounded hover:bg-primary-200`}
            onClick={onClick}
            icon={<ChevronRight className="h-4 w-4 text-base-700" />}
        />
    );

    if (tooltip) {
        drillDownButton = (
            <Tooltip content={tooltip}>
                <div className={positionClassName}>{drillDownButton}</div>
            </Tooltip>
        );
    }

    return drillDownButton;
};

DrillDownButton.propTypes = {
    onClick: PropTypes.func.isRequired,
    tooltip: PropTypes.string,
};

DrillDownButton.defaultProps = {
    tooltip: null,
};

export default DrillDownButton;
