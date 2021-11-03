import React from 'react';
import PropTypes from 'prop-types';
import { vulnerabilitySeverityLabels } from 'messages/common';

const labelClassName = 'px-2 rounded-sm p-1 border text-base';

const getSeverityClassName = (severity) => {
    switch (severity) {
        case 'LOW_VULNERABILITY_SEVERITY':
            return `${labelClassName} bg-base-200 border-base-300 text-base-800`;
        case 'MODERATE_VULNERABILITY_SEVERITY':
            return `${labelClassName} bg-warning-200 border-warning-300 text-warning-800`;
        case 'IMPORTANT_VULNERABILITY_SEVERITY':
            return `${labelClassName} bg-caution-200 border-caution-300 text-caution-800`;
        case 'CRITICAL_VULNERABILITY_SEVERITY':
            return `${labelClassName} bg-alert-200 border-alert-300 text-alert-800`;
        default:
            return '';
    }
};

const CVSSSeverityLabel = ({ severity }) => (
    <span className={getSeverityClassName(severity)}>{vulnerabilitySeverityLabels[severity]}</span>
);

CVSSSeverityLabel.propTypes = {
    severity: PropTypes.string.isRequired,
};

export default CVSSSeverityLabel;
