import { Alert, AlertVariant, Button, Modal } from '@patternfly/react-core';
import React, { ReactElement } from 'react';

export type DeleteModalProps = {
    title: string;
    isOpen: boolean;
    onClose: () => void;
    isDeleting: boolean;
    onDelete: () => void;
    error: string | null;
    children: string;
};

function DeleteModal({
    title,
    isOpen,
    onClose,
    isDeleting,
    onDelete,
    error,
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
            {error && (
                <Alert
                    isInline
                    variant={AlertVariant.danger}
                    title={error}
                    className="pf-u-mb-sm"
                />
            )}
            <p>{children}</p>
        </Modal>
    );
}

export default DeleteModal;
