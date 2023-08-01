import { Alert, AlertVariant, Button, Modal } from '@patternfly/react-core';
import React, { ReactElement } from 'react';

export type DeleteReportModalProps = {
    isOpen: boolean;
    onClose: () => void;
    isDeleting: boolean;
    onDelete: () => void;
    error: string | null;
};

function DeleteReportModal({
    isOpen,
    onClose,
    isDeleting,
    onDelete,
    error,
}: DeleteReportModalProps): ReactElement {
    return (
        <Modal
            variant="small"
            title="Permanently delete report?"
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
            <p>
                This report and any attached downloadable reports will be permanently deleted. The
                action cannot be undone.
            </p>
        </Modal>
    );
}

export default DeleteReportModal;
