import * as yup from 'yup';
import React, { ReactElement } from 'react';
import { Form, PageSection, TextArea, TextInput } from '@patternfly/react-core';
import { IntegrationFormProps } from '../integrationFormTypes';
import useIntegrationForm from '../useIntegrationForm';
import FormMessage from '../../../../Components/PatternFly/FormMessage';
import FormLabelGroup from '../FormLabelGroup';
import IntegrationFormActions from '../IntegrationFormActions';
import FormSaveButton from '../../../../Components/PatternFly/FormSaveButton';
import FormCancelButton from '../../../../Components/PatternFly/FormCancelButton';
import { CloudSourceIntegration } from '../../../../services/CloudSourceService';

export const validationSchema = yup.object().shape({
    name: yup.string().trim().required('Integration name is required'),
    type: yup.string().matches(/TYPE_PALADIN_CLOUD/),
});

export const defaultValues: CloudSourceIntegration = {
    id: '',
    name: '',
    type: 'TYPE_PALADIN_CLOUD',
    credentials: {
        secret: '',
    },
    skipTestIntegration: true,
    paladinCloud: {
        endpoint: 'https://api.paladincloud.io',
    },
};

function PaladinCloudIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<CloudSourceIntegration>): ReactElement {
    const formInitialValues = { ...defaultValues, ...initialValues };
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
    } = useIntegrationForm<CloudSourceIntegration>({
        initialValues: formInitialValues,
        validationSchema,
    });

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormLabelGroup
                        isRequired
                        label="Integration name"
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
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Paladin Cloud endpoint"
                        fieldId="paladinCloud.endpoint"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="paladinCloud.endpoint"
                            name="paladinCloud.endpoint"
                            value={values.paladinCloud?.endpoint}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Paladin Cloud token"
                        fieldId="credentials.secret"
                        touched={touched}
                        errors={errors}
                    >
                        <TextArea
                            autoResize
                            resizeOrientation="vertical"
                            isRequired
                            type="text"
                            id={`credentials.secret`}
                            value={values.credentials.secret}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                </Form>
            </PageSection>
            {isEditable && (
                <IntegrationFormActions>
                    <FormSaveButton
                        onSave={onSave}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                        isDisabled={!dirty || !isValid}
                    >
                        Save
                    </FormSaveButton>
                    <FormCancelButton onCancel={onCancel}>Cancel</FormCancelButton>
                </IntegrationFormActions>
            )}
        </>
    );
}

export default PaladinCloudIntegrationForm;
