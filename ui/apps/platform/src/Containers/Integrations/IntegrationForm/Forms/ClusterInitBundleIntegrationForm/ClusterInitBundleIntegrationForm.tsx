import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form } from '@patternfly/react-core';

import * as yup from 'yup';

import { ClusterInitBundle } from 'services/ClustersService';
import usePageState from 'Containers/Integrations/hooks/usePageState';
import NotFoundMessage from 'Components/NotFoundMessage';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import useIntegrationForm from '../../useIntegrationForm';
import IntegrationFormActions from '../../IntegrationFormActions';
import ClusterInitBundleFormMessageAlert, {
    ClusterInitBundleFormResponseMessage,
} from './ClusterInitBundleFormMessageAlert';
import FormLabelGroup from '../../FormLabelGroup';
import ClusterInitBundleDetails from './ClusterInitBundleDetails';

export type ClusterInitBundleIntegration = ClusterInitBundle;

export type ClusterInitBundleIntegrationFormValues = {
    name: string;
};

export type ClusterInitBundleIntegrationFormProps = {
    initialValues: ClusterInitBundleIntegration | null;
    isEditable?: boolean;
};

const validBundleNameRegex = /^[A-Za-z0-9._-]+$/;

export const validationSchema = yup.object().shape({
    name: yup
        .string()
        .trim()
        .required('A cluster init bundle name is required')
        .matches(
            validBundleNameRegex,
            'Name must contain only alphanumeric, ., _, or - (no spaces).'
        ),
});

export const defaultValues: ClusterInitBundleIntegrationFormValues = {
    name: '',
};

function ClusterInitBundleIntegrationForm({
    initialValues = null,
    isEditable = false,
}: ClusterInitBundleIntegrationFormProps): ReactElement {
    const formInitialValues = initialValues ? { ...initialValues, defaultValues } : defaultValues;
    const {
        values,
        touched,
        errors,
        dirty,
        isValid,
        setFieldValue,
        handleBlur,
        isSubmitting,
        isTesting,
        onSave,
        onCancel,
        message,
    } = useIntegrationForm<ClusterInitBundleIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isEditing, isViewingDetails } = usePageState();
    const isGenerated = Boolean((message as ClusterInitBundleFormResponseMessage)?.responseData);

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onSubmit(e) {
        // Press enter to submit as client-rendered form instead of server-rendered form.
        e.preventDefault();

        if (!dirty || !isValid || isSubmitting || isTesting) {
            return; // because Generate button is disabled
        }

        onSave();
    }

    // The edit flow doesn't make sense for Cluster Init Bundles so we'll show an empty state message here
    if (isEditing) {
        return (
            <NotFoundMessage
                title="This Cluster Init Bundle can not be edited"
                message="Create a new Cluster Init Bundle or delete an existing one"
            />
        );
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <div id="form-message-alert" className="pf-u-pb-md">
                    {message && <ClusterInitBundleFormMessageAlert message={message} />}
                </div>
                {isViewingDetails && initialValues ? (
                    <ClusterInitBundleDetails meta={initialValues} />
                ) : (
                    <Form isWidthLimited onSubmit={onSubmit}>
                        <FormLabelGroup
                            label="Cluster init bundle name"
                            isRequired
                            fieldId="name"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="name"
                                value={values.name}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable || isGenerated}
                            />
                        </FormLabelGroup>
                    </Form>
                )}
            </PageSection>
            {isEditable &&
                (!isGenerated ? (
                    <IntegrationFormActions>
                        <FormSaveButton
                            type="submit"
                            onSave={onSave}
                            isSubmitting={isSubmitting}
                            isTesting={isTesting}
                            isDisabled={!dirty || !isValid}
                        >
                            Generate
                        </FormSaveButton>
                        <FormCancelButton onCancel={onCancel} isDisabled={isSubmitting}>
                            Cancel
                        </FormCancelButton>
                    </IntegrationFormActions>
                ) : (
                    <IntegrationFormActions>
                        <FormCancelButton onCancel={onCancel} isDisabled={isSubmitting}>
                            Back
                        </FormCancelButton>
                    </IntegrationFormActions>
                ))}
        </>
    );
}

export default ClusterInitBundleIntegrationForm;
