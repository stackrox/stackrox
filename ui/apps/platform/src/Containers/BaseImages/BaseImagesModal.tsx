import { Alert, Button, Flex, FlexItem, Modal, Title } from '@patternfly/react-core';
import type { AxiosError } from 'axios';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type BaseImagesModalProps = {
    isOpen: boolean;
    onClose: () => void;
    onSave: () => void;
    isSuccess: boolean;
    isError: boolean;
    isSubmitting: boolean;
    error: AxiosError | null;
};

function BaseImagesModal({
    isOpen,
    onClose,
    onSave,
    isSuccess,
    isError,
    isSubmitting,
    error,
}: BaseImagesModalProps) {
    return (
        <Modal
            aria-labelledby="add-base-image-modal-title"
            header={
                <Title id="add-base-image-modal-title" headingLevel="h2">
                    Add base image path
                </Title>
            }
            isOpen={isOpen}
            onClose={onClose}
            variant="medium"
            showClose
            actions={[
                <Button
                    key="save"
                    variant="primary"
                    onClick={onSave}
                    isLoading={isSubmitting}
                    isDisabled={isSubmitting}
                >
                    Save
                </Button>,
                <Button key="cancel" variant="link" onClick={onClose}>
                    Cancel
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                {isSuccess && (
                    <FlexItem>
                        <Alert
                            variant="success"
                            isInline
                            title="Base image successfully added"
                            component="p"
                        />
                    </FlexItem>
                )}
                {isError && (
                    <FlexItem>
                        <Alert
                            variant="danger"
                            isInline
                            title="Error adding base image"
                            component="p"
                        >
                            {getAxiosErrorMessage(error)}
                        </Alert>
                    </FlexItem>
                )}
                <FlexItem>
                    <Alert variant="info" isInline title="Form coming soon" component="p">
                        Will add input for base image path (including repo path and tag pattern)
                    </Alert>
                </FlexItem>
            </Flex>
        </Modal>
    );
}

export default BaseImagesModal;
