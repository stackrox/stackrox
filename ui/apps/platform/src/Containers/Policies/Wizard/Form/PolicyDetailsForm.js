import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { reduxForm, formValueSelector, change } from 'redux-form';

import { selectors } from 'reducers';
import { isBackendFeatureFlagEnabled, knownBackendFlags } from 'utils/featureFlags';
import { getPolicyFormDataKeys } from 'Containers/Policies/Wizard/Form/utils';
import FormField from 'Components/FormField';
import Field from 'Containers/Policies/Wizard/Form/Field';
import ReduxToggleField from 'Components/forms/ReduxToggleField';
import { clientOnlyExclusionFieldNames } from './whitelistFieldNames';

function PolicyDetailsForm({
    formFields,
    includesRuntimeLifecycleStage,
    includesAuditLogEventSource,
    hasExcludedImageNames,
    changeForm,
    featureFlags,
}) {
    const auditLogEnabled = isBackendFeatureFlagEnabled(
        featureFlags,
        knownBackendFlags.ROX_K8S_AUDIT_LOG_DETECTION
    );
    return (
        <form className="flex flex-col w-full overflow-auto pb-5">
            <div className="px-3 pt-5" data-testid="policyStatusField">
                <div className="bg-base-100 border-base-200 shadow">
                    <div className="p-2 pb-2 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between leading-loose">
                        Enable Policy
                        <div className="header-control float-right">
                            <ReduxToggleField name="disabled" reverse className="self-center" />
                        </div>
                    </div>
                </div>
            </div>
            <div className="px-3 pt-5" data-testid="policyDetailsFields">
                <div className="bg-base-100 border border-base-200 shadow">
                    <div className="p-2 pb-2 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between leading-normal">
                        Policy Summary
                    </div>
                    <div className="h-full p-3">
                        {formFields.map((field) => {
                            // TODO: refactor PolicyDetailsFormFields to be iterative to avoid injecting logic in loops
                            const isEventSource = field.jsonpath === 'eventSource';
                            const isExcludedImages =
                                field.jsonpath ===
                                clientOnlyExclusionFieldNames.EXCLUDED_IMAGE_NAMES;
                            if (auditLogEnabled) {
                                // clear Event Source if Runtime lifecycle stage is not included
                                if (!includesRuntimeLifecycleStage && isEventSource) {
                                    changeForm('eventSource', undefined);
                                }
                                // clear Excluded Images if Audit Log Event Source is selected
                                if (
                                    includesRuntimeLifecycleStage &&
                                    includesAuditLogEventSource &&
                                    hasExcludedImageNames &&
                                    isExcludedImages
                                ) {
                                    changeForm(
                                        clientOnlyExclusionFieldNames.EXCLUDED_IMAGE_NAMES,
                                        []
                                    );
                                }
                            }
                            return (
                                <FormField
                                    key={field.jsonpath || field.name}
                                    label={field.label}
                                    name={field.jsonpath || field.name}
                                    required={
                                        field.required ||
                                        (isEventSource && includesRuntimeLifecycleStage)
                                    }
                                >
                                    <Field
                                        field={field}
                                        readOnly={
                                            (isEventSource && !includesRuntimeLifecycleStage) ||
                                            (isExcludedImages && includesAuditLogEventSource)
                                        }
                                    />
                                </FormField>
                            );
                        })}
                    </div>
                </div>
            </div>
        </form>
    );
}

PolicyDetailsForm.propTypes = {
    formFields: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    includesRuntimeLifecycleStage: PropTypes.bool.isRequired,
    includesAuditLogEventSource: PropTypes.bool.isRequired,
    hasExcludedImageNames: PropTypes.bool.isRequired,
    changeForm: PropTypes.func.isRequired,
    featureFlags: PropTypes.arrayOf(PropTypes.shape),
};

PolicyDetailsForm.defaultProps = {
    featureFlags: [],
};

const formFields = (state) =>
    formValueSelector('policyCreationForm')(state, ...getPolicyFormDataKeys());

const getFormData = createSelector([formFields], (formData) => formData);

const mapStateToProps = createStructuredSelector({
    includesRuntimeLifecycleStage: (state) => {
        const lifecycleStagesValue =
            formValueSelector('policyCreationForm')(state, 'lifecycleStages') || [];
        return lifecycleStagesValue.includes('RUNTIME');
    },
    includesAuditLogEventSource: (state) => {
        const eventSourceValue = formValueSelector('policyCreationForm')(state, 'eventSource');
        return eventSourceValue === 'AUDIT_LOG';
    },
    hasExcludedImageNames: (state) => {
        const excludedImageNamesValue = formValueSelector('policyCreationForm')(
            state,
            clientOnlyExclusionFieldNames.EXCLUDED_IMAGE_NAMES
        );
        return excludedImageNamesValue.length > 0;
    },
    formData: getFormData,
    featureFlags: selectors.getFeatureFlags,
});

const mapDispatchToProps = (dispatch) => ({
    changeForm: (field, value) => dispatch(change('policyCreationForm', field, value)),
});

export default reduxForm({
    form: 'policyCreationForm',
    enableReinitialize: true,
    destroyOnUnmount: false,
    keepDirtyOnReinitialize: true,
})(connect(mapStateToProps, mapDispatchToProps)(PolicyDetailsForm));
