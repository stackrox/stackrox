import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import { Modal, ModalVariant, Button } from '@patternfly/react-core';

import { resolveAlerts } from 'services/AlertsService';

type ResolveConfirmationProps = {
    resolvableAlerts: Set<string>;
    selectedAlertIds: string[];
    closeModal: () => void;
    cancelModal: () => void;
    isOpen: boolean;
};

function ResolveConfirmation({
    selectedAlertIds,
    closeModal,
    cancelModal,
    isOpen,
    resolvableAlerts,
}: ResolveConfirmationProps): ReactElement {
    function resolveAlertsAction() {
        const resolveSelection = selectedAlertIds.filter((id) => resolvableAlerts.has(id));
        resolveAlerts(resolveSelection).then(closeModal, closeModal);
    }

    const numSelectedRows = selectedAlertIds.reduce(
        (acc, id) => (resolvableAlerts.has(id) ? acc + 1 : acc),
        0
    );

    return (
        <Modal
            isOpen={isOpen}
            variant={ModalVariant.small}
            actions={[
                <Button key="confirm" variant="primary" onClick={resolveAlertsAction}>
                    Confirm
                </Button>,
                <Button key="cancel" variant="link" onClick={cancelModal}>
                    Cancel
                </Button>,
            ]}
            onClose={cancelModal}
            aria-label="Confirm resolving violations"
        >
            {`Are you sure you want to resolve ${numSelectedRows} ${pluralize(
                'violation',
                numSelectedRows
            )}?`}
        </Modal>
    );
}

export default ResolveConfirmation;
