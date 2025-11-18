import type { ReactElement } from 'react';
import { Flex, Title } from '@patternfly/react-core';
import { Modal } from '@patternfly/react-core/deprecated';

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
import type { IntegrationBase } from 'services/IntegrationsService';
import EmailNotifierForm from './EmailNotifierForm';

type FormResponseMessageWithData = FormResponseMessage & { data?: IntegrationBase };

export type EmailNotifierModalProps = {
    isOpen: boolean;
    updateNotifierList: (id: string) => void;
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

    function onSave(
        emailNotifier: EmailIntegrationFormValues
    ): Promise<FormResponseMessageWithData> {
        return createIntegration('notifiers', emailNotifier).then((integration) => {
            return { isError: false, message: '', data: integration };
        });
    }

    function onCompleteRequest(response: FormResponseMessageWithData) {
        updateNotifierList(response.data?.id ?? '');
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
