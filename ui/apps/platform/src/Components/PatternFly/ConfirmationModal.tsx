import React, { ReactElement } from 'react';
import { Modal, ModalVariant, Button } from '@patternfly/react-core';

type ConfirmationModalProps = {
    ariaLabel: string;
    closeModal: () => void;
    cancelModal: () => void;
    isOpen: boolean;
    children: ReactElement | ReactElement[] | string;
};

function ConfirmationModal({
    ariaLabel,
    closeModal,
    cancelModal,
    isOpen,
    children,
}: ConfirmationModalProps): ReactElement {
    return (
        <Modal
            isOpen={isOpen}
            variant={ModalVariant.small}
            actions={[
                <Button key="confirm" variant="primary" onClick={closeModal}>
                    Confirm
                </Button>,
                <Button key="cancel" variant="link" onClick={cancelModal}>
                    Cancel
                </Button>,
            ]}
            onClose={cancelModal}
            data-testid="delete-reports-modal"
            aria-label={ariaLabel}
        >
            {children}
        </Modal>
    );
}

export default ConfirmationModal;
