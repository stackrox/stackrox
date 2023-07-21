import React, { ReactElement } from 'react';
import { Modal, ModalVariant, Title, TitleSizes } from '@patternfly/react-core';

import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import {
    defaultValues,
    validationSchema,
    EmailIntegrationFormValues,
} from 'Containers/Integrations/IntegrationForm/Forms/EmailIntegrationForm';
import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import useFormModal from 'hooks/patternfly/useFormModal';
import { createIntegration } from 'services/IntegrationsService';
import EmailIntegrationForm from './EmailIntegrationForm';

export type EmailNotifierFormModalProps = {
    isOpen: boolean;
    updateNotifierList: (string) => void;
    onToggleEmailNotifierModal: () => void;
};

function EmailNotifierFormModal({
    isOpen,
    updateNotifierList,
    onToggleEmailNotifierModal,
}: EmailNotifierFormModalProps): ReactElement {
    const formInitialValues = { ...defaultValues };

    const { formik, message, onHandleSubmit, onHandleCancel } =
        useFormModal<EmailIntegrationFormValues>({
            initialValues: formInitialValues,
            validationSchema,
            onSendRequest: onSave,
            onCompleteRequest,
            onCancel: onToggleEmailNotifierModal,
        });

    function onSave(emailNotifier: EmailIntegrationFormValues): Promise<FormResponseMessage> {
        return createIntegration('notifiers', emailNotifier).then((response) => {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            return { isError: false, message: '', data: response?.data ?? null };
        });
    }

    function onCompleteRequest(response) {
        updateNotifierList(response?.data ?? '');
        onToggleEmailNotifierModal();
    }

    const title = 'Create email notifier';
    const descriptionText = 'Configure the setting for a new email notifier integration.';

    const header = (
        <>
            <Title id="custom-header-label" headingLevel="h1" size={TitleSizes.xl}>
                {title}
            </Title>
            <p className="pf-u-pt-sm">{descriptionText}</p>
        </>
    );

    const { values, touched, errors, isSubmitting, dirty, isValid, setFieldValue, handleBlur } =
        formik;

    return (
        <Modal
            aria-label="Create new email notifier"
            variant={ModalVariant.medium}
            header={header}
            isOpen={isOpen}
            onClose={onHandleCancel}
            actions={[
                <FormSaveButton
                    onSave={onHandleSubmit}
                    isSubmitting={isSubmitting}
                    isTesting={false}
                    isDisabled={!dirty || !isValid}
                >
                    Save integration
                </FormSaveButton>,
                <FormCancelButton onCancel={onHandleCancel}>Cancel</FormCancelButton>,
            ]}
        >
            <FormMessage message={message} />
            <EmailIntegrationForm
                values={values}
                setFieldValue={setFieldValue}
                handleBlur={handleBlur}
                touched={touched}
                errors={errors}
            />
        </Modal>
    );
}

export default EmailNotifierFormModal;
