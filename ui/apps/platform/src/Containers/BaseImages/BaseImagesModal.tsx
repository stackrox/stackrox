import { Button, Modal, Title } from '@patternfly/react-core';
import BaseImagesForm from './BaseImagesForm';
import type { BaseImagesFormProps } from './BaseImagesForm';

export type BaseImagesModalProps = {
    isOpen: boolean;
    onClose: () => void;
    formProps: BaseImagesFormProps;
};

function BaseImagesModal({ isOpen, onClose, formProps }: BaseImagesModalProps) {
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
            <BaseImagesForm {...formProps} />
        </Modal>
    );
}

export default BaseImagesModal;
