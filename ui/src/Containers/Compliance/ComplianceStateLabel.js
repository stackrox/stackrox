import React from 'react';
import PropTypes from 'prop-types';

const COMPLIANCE_STATES = {
    COMPLIANCE_STATE_UNKNOWN: 'COMPLIANCE_STATE_UNKNOWN',
    COMPLIANCE_STATE_SKIP: 'COMPLIANCE_STATE_SKIP',
    COMPLIANCE_STATE_NOTE: 'COMPLIANCE_STATE_NOTE',
    COMPLIANCE_STATE_SUCCESS: 'COMPLIANCE_STATE_SUCCESS',
    COMPLIANCE_STATE_FAILURE: 'COMPLIANCE_STATE_FAILURE',
    COMPLIANCE_STATE_ERROR: 'COMPLIANCE_STATE_ERROR'
};

const COMPLIANCE_STATE_VALUES = Object.values(COMPLIANCE_STATES);

const STATE_LABEL = {
    COMPLIANCE_STATE_UNKNOWN: 'Unknown',
    COMPLIANCE_STATE_SKIP: 'Skip',
    COMPLIANCE_STATE_NOTE: 'Note',
    COMPLIANCE_STATE_SUCCESS: 'Success',
    COMPLIANCE_STATE_FAILURE: 'Failure',
    COMPLIANCE_STATE_ERROR: 'Error'
};

const getClassName = state => {
    let className = '';
    switch (state) {
        case COMPLIANCE_STATES.COMPLIANCE_STATE_SUCCESS:
            className = 'bg-success-300 text-success-900';
            break;
        case COMPLIANCE_STATES.COMPLIANCE_STATE_FAILURE:
        case COMPLIANCE_STATES.COMPLIANCE_STATE_ERROR:
            className = 'bg-alert-300 text-alert-900';
            break;
        default:
            className = 'bg-base-300 text-base-900';
            break;
    }
    return className;
};

const ComplianceStateLabel = ({ state }) => (
    <span className={`py-1 px-2 rounded ${getClassName(state)}`}>{STATE_LABEL[state]}</span>
);

ComplianceStateLabel.propTypes = {
    state: PropTypes.oneOf(COMPLIANCE_STATE_VALUES).isRequired
};

export default ComplianceStateLabel;
