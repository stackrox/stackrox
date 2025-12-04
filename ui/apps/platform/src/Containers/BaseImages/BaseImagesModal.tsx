import { Alert, Button, Modal, Title } from '@patternfly/react-core';
import type { AxiosError } from 'axios';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import BaseImagesForm from './BaseImagesForm';
import type { BaseImagesFormProps } from './BaseImagesForm';

export type BaseImagesModalProps = {
    isOpen: boolean;
    onClose: () => void;
    isSuccess: boolean;
    isError: boolean;
    error: AxiosError | null;
    formProps: BaseImagesFormProps;
};

function BaseImagesModal({
    isOpen,
    onClose,
    isSuccess,
    isError,
    error,
    formProps,
}: BaseImagesModalProps) {
    return (
        <Modal
            aria-labelledby="add-base-image-modal-title"
            header={
                <Title id="add-base-image-modal-title" headingLevel="h2" size="2xl">
                    Add base image
                </Title>
            }
            isOpen={isOpen}
            onClose={onClose}
            variant="medium"
            showClose={false}
            actions={[
                <Button key="Close" onClick={onClose} isDisabled={formProps.isSubmitting}>
                    Close
                </Button>,
            ]}
        >
            {isSuccess && (
                <Alert
                    variant="success"
                    isInline
                    title="Base image successfully added"
                    component="p"
                />
            )}
            {isError && (
                <Alert variant="danger" isInline title="Error adding base image" component="p">
                    {getAxiosErrorMessage(error)}
                </Alert>
            )}
            <BaseImagesForm {...formProps} />
        </Modal>
    );
}

export default BaseImagesModal;
