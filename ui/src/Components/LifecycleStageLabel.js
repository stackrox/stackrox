import React from 'react';
import PropTypes from 'prop-types';
import { lifecycleStageLabels } from 'messages/common';

const labelClassName = 'px-2 rounded-full border-2';

const getLifecycleClassName = lifecycleStage => {
    switch (lifecycleStage) {
        case 'BUILD':
            return `${labelClassName} bg-primary-200 border-primary-300 text-primary-800`;
        case 'DEPLOY':
            return `${labelClassName} bg-secondary-200 border-secondary-300 text-secondary-800`;
        case 'RUNTIME':
            return `${labelClassName} bg-tertiary-200 border-tertiary-300 text-tertiary-800`;
        default:
            return '';
    }
};

const LifecycleStageLabel = ({ className, lifecycleStage }) => {
    return (
        <span className={`${getLifecycleClassName(lifecycleStage)} ${className}`}>
            {lifecycleStageLabels[lifecycleStage]}
        </span>
    );
};

LifecycleStageLabel.propTypes = {
    className: PropTypes.string,
    lifecycleStage: PropTypes.string.isRequired
};

LifecycleStageLabel.defaultProps = {
    className: ''
};

export default LifecycleStageLabel;
