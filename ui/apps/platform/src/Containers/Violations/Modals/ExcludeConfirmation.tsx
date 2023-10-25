import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import { Modal, ModalVariant, Button } from '@patternfly/react-core';

import { excludeDeployments } from 'services/PoliciesService';
import { ListAlert } from 'types/alert.proto';

// Filter the excludableAlerts displayed down to the ones checked, and group them into a map from policy ID to a list of
// deployment names, then exclude every policy ID, deployment name pair in the map.
function excludeAlerts(checkedAlertIds, excludableAlerts) {
    const checkedAlertsSet = new Set(checkedAlertIds);

    const policyToDeployments = {};
    excludableAlerts
        .filter(({ id }) => checkedAlertsSet.has(id))
        .forEach(({ policy, deployment }) => {
            if (!policyToDeployments[policy.id]) {
                policyToDeployments[policy.id] = [deployment.name];
            } else {
                policyToDeployments[policy.id].push(deployment.name);
            }
        });

    return Promise.all(
        Object.keys(policyToDeployments).map((policyId) => {
            return excludeDeployments(policyId, policyToDeployments[policyId]);
        })
    );
}

type ExcludeConfirmationProps = {
    excludableAlerts: ListAlert[];
    selectedAlertIds: string[];
    closeModal: () => void;
    cancelModal: () => void;
    isOpen: boolean;
};

function ExcludeConfirmation({
    excludableAlerts,
    selectedAlertIds,
    closeModal,
    cancelModal,
    isOpen,
}: ExcludeConfirmationProps): ReactElement {
    function excludeDeploymentsAction() {
        excludeAlerts(selectedAlertIds, excludableAlerts).then(closeModal, closeModal);
    }

    const numSelectedRows = selectedAlertIds.length;
    return (
        <Modal
            isOpen={isOpen}
            variant={ModalVariant.small}
            actions={[
                <Button key="confirm" variant="primary" onClick={excludeDeploymentsAction}>
                    Confirm
                </Button>,
                <Button key="cancel" variant="link" onClick={cancelModal}>
                    Cancel
                </Button>,
            ]}
            onClose={cancelModal}
            aria-label="Confirm excluding violations"
        >
            {`Are you sure you want to exclude ${numSelectedRows} ${pluralize(
                'violation',
                numSelectedRows
            )}?`}
        </Modal>
    );
}

export default ExcludeConfirmation;
