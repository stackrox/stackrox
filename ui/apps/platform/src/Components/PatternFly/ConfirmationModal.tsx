import React, { ReactElement, ReactNode } from 'react';
import { Modal, ModalVariant, Button, ButtonVariant } from '@patternfly/react-core';

type ConfirmationModalProps = {
    ariaLabel: string;
    confirmText: string;
    onConfirm: () => void;
    onCancel: () => void;
    isOpen: boolean;
    isLoading?: boolean; // if modal remains open until finally block of request promise
    children: ReactNode;
};

function ConfirmationModal({
    ariaLabel,
    confirmText,
    onConfirm,
    onCancel,
    isOpen,
    isLoading,
    children,
}: ConfirmationModalProps): ReactElement {
    return (
        <Modal
            isOpen={isOpen}
            variant={ModalVariant.small}
            actions={[
                <Button
                    key="confirm"
                    variant={ButtonVariant.danger}
                    onClick={onConfirm}
                    isDisabled={isLoading}
                    isLoading={isLoading}
                >
                    {confirmText}
                </Button>,
                <Button key="cancel" variant="link" onClick={onCancel} isDisabled={isLoading}>
                    Cancel
                </Button>,
            ]}
            onClose={onCancel}
            data-testid="confirmation-modal"
            aria-label={ariaLabel}
        >
            {children}
        </Modal>
    );
}

export default ConfirmationModal;
