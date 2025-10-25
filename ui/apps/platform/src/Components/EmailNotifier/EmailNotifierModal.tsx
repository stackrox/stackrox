import type { ReactElement } from 'react';
import { Flex, Modal, Title } from '@patternfly/react-core';

import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import {
    defaultValues,
    validationSchema,
} from 'Containers/Integrations/IntegrationForm/Forms/EmailIntegrationForm';
import type { EmailIntegrationFormValues } from 'Containers/Integrations/IntegrationForm/Forms/EmailIntegrationForm';
import FormMessage from 'Components/PatternFly/FormMessage';
import type { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import useFormModal from 'hooks/patternfly/useFormModal';
import { createIntegration } from 'services/IntegrationsService';
import EmailNotifierForm from './EmailNotifierForm';

export type EmailNotifierModalProps = {
    isOpen: boolean;
    updateNotifierList: (string) => void;
    onToggleEmailNotifierModal: () => void;
};

function EmailNotifierModal({
    isOpen,
    updateNotifierList,
    onToggleEmailNotifierModal,
}: EmailNotifierModalProps): ReactElement {
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
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
            <Title id="custom-header-label" headingLevel="h1">
                {title}
            </Title>
            <p>{descriptionText}</p>
        </Flex>
    );

    const { values, touched, errors, isSubmitting, dirty, isValid, setFieldValue, handleBlur } =
        formik;

    return (
        <Modal
            aria-label="Create new email notifier"
            variant="medium"
            header={header}
            isOpen={isOpen}
            onClose={onHandleCancel}
            actions={[
                <FormSaveButton
                    key="save"
                    onSave={onHandleSubmit}
                    isSubmitting={isSubmitting}
                    isTesting={false}
                    isDisabled={!dirty || !isValid}
                >
                    Save integration
                </FormSaveButton>,
                <FormCancelButton key="cancel" onCancel={onHandleCancel}>
                    Cancel
                </FormCancelButton>,
            ]}
        >
            <FormMessage message={message} />
            <EmailNotifierForm
                values={values}
                setFieldValue={setFieldValue}
                handleBlur={handleBlur}
                touched={touched}
                errors={errors}
            />
        </Modal>
    );
}

export default EmailNotifierModal;
