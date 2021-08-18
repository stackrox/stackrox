import React, { useEffect } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { reduxForm, formValueSelector, change, FieldArray } from 'redux-form';

import { selectors } from 'reducers';
import { lifecycleStageLabels, severityLabels } from 'messages/common';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import FormField from 'Components/FormField';
import ReduxToggleField from 'Components/forms/ReduxToggleField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxMultiSelectField from 'Components/forms/ReduxMultiSelectField';
import ReduxTextAreaField from 'Components/forms/ReduxTextAreaField';
import ReduxMultiSelectCreatableField from 'Components/forms/ReduxMultiSelectCreatableField';
import RestrictToScope from './RestrictToScope';
import ExcludedScope from './ExcludedScope';
import { clientOnlyExclusionFieldNames } from '../whitelistFieldNames';
import { FormSection, FormSectionBody } from './FormSection';
import MitreAttackVectorBuilder from './MitreAttackVectorBuilder';

function filterEventSourceOptions(option) {
    return option.value !== 'NOT_APPLICABLE';
}
function PolicyDetailsForm({
    includesRuntimeLifecycleStage,
    includesAuditLogEventSource,
    hasExcludedImageNames,
    changeForm,
    notifiers,
    images,
    policyCategories,
    mitreVectorsLocked,
}) {
    const auditLogEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_K8S_AUDIT_LOG_DETECTION);
    const isMitreEnabled = useFeatureFlagEnabled(
        knownBackendFlags.ROX_SYSTEM_POLICY_MITRE_FRAMEWORK
    );
    useEffect(() => {
        if (auditLogEnabled) {
            // clear Event Source if Runtime lifecycle stage is not included
            if (!includesRuntimeLifecycleStage) {
                changeForm('eventSource', 'NOT_APPLICABLE');
            }
            // clear Excluded Images if Audit Log Event Source is selected
            if (
                includesRuntimeLifecycleStage &&
                includesAuditLogEventSource &&
                hasExcludedImageNames
            ) {
                changeForm(clientOnlyExclusionFieldNames.EXCLUDED_IMAGE_NAMES, []);
            }
        }
    }, [
        auditLogEnabled,
        includesAuditLogEventSource,
        includesRuntimeLifecycleStage,
        hasExcludedImageNames,
        changeForm,
    ]);

    return (
        <form className="flex flex-col w-full overflow-auto gap-4 p-4">
            <FormSection dataTestId="policyStatusField">
                <FormSectionBody>
                    <FormField label="Enabled" isInline noMargin>
                        <ReduxToggleField name="disabled" reverse className="self-center" />
                    </FormField>
                </FormSectionBody>
            </FormSection>
            <FormSection dataTestId="policyDetailsFields" headerText="Policy Summary">
                <FormSectionBody>
                    <FormField label="Name" required>
                        <ReduxTextField name="name" />
                    </FormField>
                    <FormField label="Severity" required>
                        <ReduxSelectField
                            name="severity"
                            options={Object.keys(severityLabels).map((key) => ({
                                label: severityLabels[key],
                                value: key,
                            }))}
                            placeholder="Select a severity level"
                        />
                    </FormField>
                    <FormField label="Lifecycle Stages" required testId="lifecycle-stages">
                        <ReduxMultiSelectField
                            name="lifecycleStages"
                            options={Object.keys(lifecycleStageLabels).map((key) => ({
                                label: lifecycleStageLabels[key],
                                value: key,
                            }))}
                        />
                    </FormField>
                    {auditLogEnabled && (
                        <FormField
                            label="Event Sources"
                            required={includesRuntimeLifecycleStage}
                            testId="event-sources"
                        >
                            <ReduxSelectField
                                name="eventSource"
                                options={[
                                    {
                                        label: 'Not applicable to selected lifecycle ',
                                        value: 'NOT_APPLICABLE',
                                    },
                                    {
                                        label: 'Deployment',
                                        value: 'DEPLOYMENT_EVENT',
                                    },
                                    { label: 'Audit Log', value: 'AUDIT_LOG_EVENT' },
                                ]}
                                disabled={!includesRuntimeLifecycleStage}
                                filterOption={filterEventSourceOptions}
                            />
                        </FormField>
                    )}
                    <FormField label="Description">
                        <ReduxTextAreaField
                            name="description"
                            placeholder="What does this policy do?"
                        />
                    </FormField>
                    <FormField label="Rationale">
                        <ReduxTextAreaField
                            name="rationale"
                            placeholder="Why does this policy exist?"
                        />
                    </FormField>
                    <FormField label="Remediation">
                        <ReduxTextAreaField
                            name="remediation"
                            placeholder="What can an operator do to resolve any violations?"
                        />
                    </FormField>
                    <FormField label="Categories" required>
                        <ReduxMultiSelectCreatableField
                            name="categories"
                            options={policyCategories.map((category) => ({
                                label: category,
                                value: category,
                            }))}
                        />
                    </FormField>
                    <FormField label="Notifications">
                        <ReduxMultiSelectField
                            name="notifiers"
                            options={notifiers.map((notifier) => ({
                                label: notifier.name,
                                value: notifier.id,
                            }))}
                        />
                    </FormField>
                    <FormField label="Restrict to Scope" testId="restrict-to-scope">
                        <FieldArray name="scope" component={RestrictToScope} />
                    </FormField>
                    <FormField label="Exclude by Scope" testId="exclude-by-scope">
                        <FieldArray
                            name={clientOnlyExclusionFieldNames.EXCLUDED_DEPLOYMENT_SCOPES}
                            component={ExcludedScope}
                        />
                    </FormField>
                    <FormField
                        label="Excluded Images (Build Lifecycle only)"
                        testId="excluded-images"
                    >
                        <ReduxMultiSelectCreatableField
                            name={clientOnlyExclusionFieldNames.EXCLUDED_IMAGE_NAMES}
                            options={images.map((image) => ({
                                label: image.name,
                                value: image.name,
                            }))}
                            disabled={auditLogEnabled && includesAuditLogEventSource}
                        />
                    </FormField>
                </FormSectionBody>
            </FormSection>
            {isMitreEnabled && (
                <FormSection dataTestId="mitreAttackVectorFormFields" headerText="MITRE ATT&CK">
                    <FieldArray
                        name="mitreAttackVectors"
                        component={MitreAttackVectorBuilder}
                        rerenderOnEveryChange
                        props={{
                            isReadOnly: mitreVectorsLocked,
                        }}
                    />
                </FormSection>
            )}
        </form>
    );
}

PolicyDetailsForm.propTypes = {
    includesRuntimeLifecycleStage: PropTypes.bool.isRequired,
    includesAuditLogEventSource: PropTypes.bool.isRequired,
    hasExcludedImageNames: PropTypes.bool.isRequired,
    changeForm: PropTypes.func.isRequired,
    policyCategories: PropTypes.arrayOf(PropTypes.string),
    images: PropTypes.arrayOf(PropTypes.shape({})),
    notifiers: PropTypes.arrayOf(PropTypes.shape({})),
    mitreVectorsLocked: PropTypes.bool,
};

PolicyDetailsForm.defaultProps = {
    policyCategories: [],
    images: [],
    notifiers: [],
    mitreVectorsLocked: false,
};

const mapStateToProps = createStructuredSelector({
    includesRuntimeLifecycleStage: (state) => {
        const lifecycleStagesValue =
            formValueSelector('policyCreationForm')(state, 'lifecycleStages') || [];
        return lifecycleStagesValue.includes('RUNTIME');
    },
    includesAuditLogEventSource: (state) => {
        const eventSourceValue = formValueSelector('policyCreationForm')(state, 'eventSource');
        return eventSourceValue === 'AUDIT_LOG_EVENT';
    },
    hasExcludedImageNames: (state) => {
        const excludedImageNamesValue = formValueSelector('policyCreationForm')(
            state,
            clientOnlyExclusionFieldNames.EXCLUDED_IMAGE_NAMES
        );
        return excludedImageNamesValue.length > 0;
    },
    notifiers: selectors.getNotifiers,
    images: selectors.getImages,
    policyCategories: selectors.getPolicyCategories,
    mitreVectorsLocked: (state) => {
        const mitreVectorsLocked = formValueSelector('policyCreationForm')(
            state,
            'mitreVectorsLocked'
        );
        return mitreVectorsLocked;
    },
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
