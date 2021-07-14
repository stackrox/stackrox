import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { FieldArray, reduxForm, formValueSelector } from 'redux-form';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import PolicyBuilderKeys from './PolicyBuilderKeys';
import PolicySections from './PolicySections';
import {
    policyConfigurationDescriptor,
    networkDetectionDescriptor,
    auditLogDescriptor,
} from './descriptors';

function BooleanPolicySection({ readOnly, hasHeader, hasAuditLogEventSource }) {
    const [descriptor, setDescriptor] = useState([]);
    const networkDetectionBaselineViolationEnabled = useFeatureFlagEnabled(
        knownBackendFlags.ROX_NETWORK_DETECTION_BASELINE_VIOLATION
    );
    const auditLogEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_K8S_AUDIT_LOG_DETECTION);

    useEffect(() => {
        if (auditLogEnabled && hasAuditLogEventSource) {
            setDescriptor(auditLogDescriptor);
        } else if (networkDetectionBaselineViolationEnabled) {
            setDescriptor([...policyConfigurationDescriptor, ...networkDetectionDescriptor]);
        } else {
            setDescriptor(policyConfigurationDescriptor);
        }
    }, [auditLogEnabled, hasAuditLogEventSource, networkDetectionBaselineViolationEnabled]);
    if (readOnly) {
        return (
            <div className="w-full flex">
                {descriptor.length > 0 && (
                    <FieldArray
                        name="policySections"
                        component={PolicySections}
                        hasHeader={hasHeader}
                        readOnly
                        className="w-full"
                        descriptor={descriptor}
                    />
                )}
            </div>
        );
    }
    return (
        <DndProvider backend={HTML5Backend}>
            <div className="w-full h-full flex">
                {descriptor.length > 0 && (
                    <>
                        <FieldArray
                            name="policySections"
                            component={PolicySections}
                            descriptor={descriptor}
                            hasAuditLogEventSource={hasAuditLogEventSource}
                        />
                        <PolicyBuilderKeys keys={descriptor} />
                    </>
                )}
            </div>
        </DndProvider>
    );
}

BooleanPolicySection.propTypes = {
    readOnly: PropTypes.bool,
    hasHeader: PropTypes.bool,
    hasAuditLogEventSource: PropTypes.bool.isRequired,
};

BooleanPolicySection.defaultProps = {
    readOnly: false,
    hasHeader: true,
};

const mapStateToProps = createStructuredSelector({
    hasAuditLogEventSource: (state) => {
        const eventSourceValue = formValueSelector('policyCreationForm')(state, 'eventSource');
        return eventSourceValue === 'AUDIT_LOG_EVENT';
    },
});

export default reduxForm({
    form: 'policyCreationForm',
    enableReinitialize: true,
    immutableProps: ['initialValues'],
    destroyOnUnmount: false,
})(connect(mapStateToProps, null)(BooleanPolicySection));
