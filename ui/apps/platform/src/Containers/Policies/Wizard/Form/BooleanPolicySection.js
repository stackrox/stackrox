import React from 'react';
import PropTypes from 'prop-types';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { FieldArray, reduxForm } from 'redux-form';
import { connect } from 'react-redux';

import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import PolicyBuilderKeys from 'Components/PolicyBuilderKeys';
import PolicySections from './PolicySections';
import { getPolicyConfiguration } from './descriptors';

function BooleanPolicySection({ readOnly, hasHeader }) {
    const k8sEventsEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_K8S_EVENTS_DETECTION);
    const featureFlags = {
        [knownBackendFlags.ROX_K8S_EVENTS_DETECTION]: k8sEventsEnabled,
    };
    const keys = getPolicyConfiguration(featureFlags).descriptor;
    if (readOnly) {
        return (
            <div className="w-full flex">
                <FieldArray
                    name="policySections"
                    component={PolicySections}
                    hasHeader={hasHeader}
                    readOnly
                    className="w-full"
                />
            </div>
        );
    }
    return (
        <DndProvider backend={HTML5Backend}>
            <div className="w-full h-full flex">
                <FieldArray name="policySections" component={PolicySections} />
                <PolicyBuilderKeys keys={keys} />
            </div>
        </DndProvider>
    );
}

BooleanPolicySection.propTypes = {
    readOnly: PropTypes.bool,
    hasHeader: PropTypes.bool,
};

BooleanPolicySection.defaultProps = {
    readOnly: false,
    hasHeader: true,
};

export default reduxForm({
    form: 'policyCreationForm',
    enableReinitialize: true,
    immutableProps: ['initialValues'],
    destroyOnUnmount: false,
})(connect(null)(BooleanPolicySection));
