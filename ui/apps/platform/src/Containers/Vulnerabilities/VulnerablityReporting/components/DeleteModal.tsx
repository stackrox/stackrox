import { Button, Modal } from '@patternfly/react-core';
import React, { ReactElement } from 'react';

export type DeleteModalProps = {
    title: string;
    isOpen: boolean;
    onClose: () => void;
    isDeleting: boolean;
    onDelete: () => void;
    children: ReactElement | ReactElement[];
};

function DeleteModal({
    title,
    isOpen,
    onClose,
    isDeleting,
    onDelete,
    children,
}: DeleteModalProps): ReactElement {
    return (
        <Modal
            variant="small"
            title={title}
            isOpen={isOpen}
            onClose={onClose}
            actions={[
                <Button
                    key="confirm"
                    variant="danger"
                    isLoading={isDeleting}
                    isDisabled={isDeleting}
                    onClick={onDelete}
                >
                    Delete
                </Button>,
                <Button key="cancel" variant="secondary" onClick={onClose}>
                    Cancel
                </Button>,
            ]}
        >
            {children}
        </Modal>
    );
}

export default DeleteModal;
