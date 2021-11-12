import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import { Modal, ModalVariant, Button } from '@patternfly/react-core';

type DeleteConfirmationProps = {
    selectedReportIds: string[];
    closeModal: () => void;
    cancelModal: () => void;
    isOpen: boolean;
};

function DeleteConfirmation({
    selectedReportIds,
    closeModal,
    cancelModal,
    isOpen,
}: DeleteConfirmationProps): ReactElement {
    const numSelectedRows = selectedReportIds.length;

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
            aria-label="Confirm deleting reports"
        >
            {`Are you sure you want to delete ${numSelectedRows} ${pluralize(
                'report',
                numSelectedRows
            )}?`}
        </Modal>
    );
}

export default DeleteConfirmation;
