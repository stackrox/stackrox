import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { FieldArray, reduxForm, formValueSelector } from 'redux-form';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { Message } from '@stackrox/ui-components';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import PolicyBuilderKeys from './PolicyBuilderKeys';
import PolicySections from './PolicySections';
import {
    policyConfigurationDescriptor,
    networkDetectionDescriptor,
    auditLogDescriptor,
    imageSigningCriteriaDescriptor,
    networkPolicyFieldDescriptors,
} from './descriptors';

function BooleanPolicySection({ readOnly, hasHeader, hasAuditLogEventSource, criteriaLocked }) {
    const [descriptor, setDescriptor] = useState([]);
    const isImageSigningEnabled = useFeatureFlagEnabled('ROX_VERIFY_IMAGE_SIGNATURE');
    const isNetworkPolicyFieldsEnabled = useFeatureFlagEnabled('ROX_NETPOL_FIELDS');
    useEffect(() => {
        if (hasAuditLogEventSource) {
            setDescriptor(auditLogDescriptor);
        } else {
            setDescriptor([
                ...policyConfigurationDescriptor,
                ...networkDetectionDescriptor,
                ...(isImageSigningEnabled ? [imageSigningCriteriaDescriptor] : []),
                ...(isNetworkPolicyFieldsEnabled ? networkPolicyFieldDescriptors : []),
            ]);
        }
    }, [hasAuditLogEventSource, isImageSigningEnabled, isNetworkPolicyFieldsEnabled]);

    if (readOnly || criteriaLocked) {
        return (
            <div>
                {criteriaLocked && !readOnly && (
                    <div className="p-4">
                        <Message>
                            Editing Policy Criteria is disabled for default system policies. Please
                            clone this system policy or create a new system policy to perform edits
                            to Policy Criteria.
                        </Message>
                    </div>
                )}
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
    criteriaLocked: PropTypes.bool,
};

BooleanPolicySection.defaultProps = {
    readOnly: false,
    hasHeader: true,
    criteriaLocked: false,
};

const mapStateToProps = createStructuredSelector({
    hasAuditLogEventSource: (state) => {
        const eventSourceValue = formValueSelector('policyCreationForm')(state, 'eventSource');
        return eventSourceValue === 'AUDIT_LOG_EVENT';
    },
    criteriaLocked: (state) => {
        const criteriaLocked = formValueSelector('policyCreationForm')(state, 'criteriaLocked');
        return criteriaLocked;
    },
});

export default reduxForm({
    form: 'policyCreationForm',
    enableReinitialize: true,
    immutableProps: ['initialValues'],
    destroyOnUnmount: false,
})(connect(mapStateToProps, null)(BooleanPolicySection));
