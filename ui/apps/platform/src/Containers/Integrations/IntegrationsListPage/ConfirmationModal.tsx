import React, { ReactElement, ReactNode } from 'react';
import { Modal, ModalVariant, Button } from '@patternfly/react-core';

export type DeleteIntegrationsConfirmationProps = {
    isOpen: boolean;
    children: ReactNode;
    onCancel: () => void;
    onConfirm: () => void;
};

function DeleteIntegrationsConfirmation({
    isOpen,
    children,
    onCancel,
    onConfirm,
}: DeleteIntegrationsConfirmationProps): ReactElement {
    return (
        <Modal
            variant={ModalVariant.small}
            isOpen={isOpen}
            onClose={onCancel}
            actions={[
                <Button key="confirm" variant="primary" onClick={onConfirm}>
                    Confirm
                </Button>,
                <Button key="cancel" variant="link" onClick={onCancel}>
                    Cancel
                </Button>,
            ]}
        >
            {children}
        </Modal>
    );
}

export default DeleteIntegrationsConfirmation;
