import React, { ReactElement, ReactNode } from 'react';
import { Modal, ModalVariant, Button, ButtonVariant } from '@patternfly/react-core';

type ConfirmationModalProps = {
    ariaLabel: string;
    title?: string;
    confirmText: string;
    onConfirm: () => void;
    onCancel: () => void;
    isConfirmDisabled?: boolean;
    isOpen: boolean;
    isLoading?: boolean; // if modal remains open until finally block of request promise
    isDestructive?: boolean;
    children: ReactNode;
};

function ConfirmationModal({
    ariaLabel,
    title,
    confirmText,
    onConfirm,
    onCancel,
    isConfirmDisabled = false,
    isOpen,
    isLoading,
    isDestructive = true,
    children,
}: ConfirmationModalProps): ReactElement {
    return (
        <Modal
            isOpen={isOpen}
            variant={ModalVariant.small}
            title={title || ''}
            titleIconVariant="warning"
            actions={[
                <Button
                    key="confirm"
                    variant={isDestructive ? ButtonVariant.danger : ButtonVariant.primary}
                    onClick={onConfirm}
                    className="pf-confirmation-modal-confirm-btn"
                    isDisabled={isConfirmDisabled || isLoading}
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
