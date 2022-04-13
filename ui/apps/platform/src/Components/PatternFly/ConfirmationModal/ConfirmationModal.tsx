import React, { ReactElement, ReactNode } from 'react';
import { Modal, ModalVariant, Button, ButtonVariant } from '@patternfly/react-core';

import './ConfirmationModal.css';

type ConfirmationModalProps = {
    ariaLabel: string;
    title?: string;
    confirmText: string;
    onConfirm: () => void;
    onCancel: () => void;
    isOpen: boolean;
    isLoading?: boolean; // if modal remains open until finally block of request promise
    isDestructiveAction?: boolean;
    children: ReactNode;
};

function ConfirmationModal({
    ariaLabel,
    title,
    confirmText,
    onConfirm,
    onCancel,
    isOpen,
    isLoading,
    isDestructiveAction = true,
    children,
}: ConfirmationModalProps): ReactElement {
    return (
        <Modal
            isOpen={isOpen}
            variant={ModalVariant.small}
            title={title || ''}
            actions={[
                <Button
                    key="confirm"
                    variant={isDestructiveAction ? ButtonVariant.danger : ButtonVariant.primary}
                    onClick={onConfirm}
                    className="pf-confirmation-modal-confirm-btn"
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
