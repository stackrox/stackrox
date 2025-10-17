import type { ReactElement } from 'react';
import pluralize from 'pluralize';
import { Alert, Button, Flex, Modal, Text } from '@patternfly/react-core';

import { excludeDeployments } from 'services/PoliciesService';
import type { DeploymentListAlert, ListAlert } from 'types/alert.proto';
import useRestMutation from 'hooks/useRestMutation';
import type { Empty } from 'services/types';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

// Filter the excludableAlerts displayed down to the ones checked, and group them into a map from policy ID to a list of
// deployment names, then exclude every policy ID, deployment name pair in the map.
function excludeAlerts({
    selectedAlertIds,
    excludableAlerts,
}: {
    selectedAlertIds: string[];
    excludableAlerts: ListAlert[];
}): Promise<Empty[]> {
    const checkedAlertsSet = new Set(selectedAlertIds);

    const policyToDeployments = {};
    excludableAlerts
        .filter(({ id }) => checkedAlertsSet.has(id))
        .filter((alert): alert is DeploymentListAlert => 'deployment' in alert)
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
    const { mutate, isLoading, error, reset } = useRestMutation(excludeAlerts, {
        onSuccess: () => {
            closeModal();
            reset();
        },
    });

    const numSelectedRows = selectedAlertIds.length;
    return (
        <Modal
            isOpen={isOpen}
            variant="medium"
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    isDisabled={isLoading}
                    isLoading={isLoading}
                    onClick={() => mutate({ selectedAlertIds, excludableAlerts })}
                >
                    Confirm
                </Button>,
                <Button key="cancel" variant="link" onClick={cancelModal}>
                    Cancel
                </Button>,
            ]}
            onClose={() => {
                cancelModal();
                reset();
            }}
            aria-label="Confirm excluding violations"
        >
            <Flex direction={{ default: 'column' }}>
                <Text>
                    {`Are you sure you want to exclude deployments from ${numSelectedRows} policy ${pluralize(
                        'violation',
                        numSelectedRows
                    )}?`}
                </Text>
                {!!error && (
                    <Alert
                        variant="danger"
                        title="There was an error excluding one or more deployments"
                        component="p"
                        isInline
                    >
                        {getAxiosErrorMessage(error)}
                    </Alert>
                )}
            </Flex>
        </Modal>
    );
}

export default ExcludeConfirmation;
