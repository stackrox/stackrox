import React, { ReactElement, useState } from 'react';
import {
    Alert,
    Button,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    Modal,
} from '@patternfly/react-core';

import useAnalytics, { REVOKE_CLUSTER_REGISTRATION_SECRET } from 'hooks/useAnalytics';
import useRestMutation from 'hooks/useRestMutation';
import {
    ClusterRegistrationSecret,
    revokeClusterRegistrationSecrets,
} from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type RevokeClusterRegistrationSecretModalProps = {
    clusterRegistrationSecret: ClusterRegistrationSecret;
    onCloseModal: (wasRevoked: boolean) => void;
};

function RevokeClusterRegistrationSecretModal({
    clusterRegistrationSecret,
    onCloseModal,
}: RevokeClusterRegistrationSecretModalProps): ReactElement {
    const { analyticsTrack } = useAnalytics();
    const { mutate, isLoading, error } = useRestMutation(
        (ids: string[]) => revokeClusterRegistrationSecrets(ids),
        {
            onSuccess: ({ crsRevocationErrors }) => {
                if (crsRevocationErrors.length === 0) {
                    onCloseModal(true);
                }
                analyticsTrack(REVOKE_CLUSTER_REGISTRATION_SECRET);
            },
        }
    );

    function onRevokeClusterRegistrationSecret() {
        mutate([clusterRegistrationSecret.id]);
    }

    function onCancel() {
        onCloseModal(false);
    }

    // showClose={false} to prevent clicking close while isRevokingClusterRegistrationSecret.
    return (
        <Modal
            title="Revoke cluster registration secret"
            variant="small"
            isOpen
            showClose={false}
            actions={[
                <Button
                    key="Revoke cluster registration secret"
                    variant="primary"
                    onClick={onRevokeClusterRegistrationSecret}
                    isDisabled={isLoading}
                >
                    Revoke cluster registration secret
                </Button>,
                <Button key="Cancel" variant="secondary" onClick={onCancel} isDisabled={isLoading}>
                    Cancel
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                <DescriptionList isHorizontal>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Name</DescriptionListTerm>
                        <DescriptionListDescription>
                            {clusterRegistrationSecret.name}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
                {error !== undefined && (
                    <Alert
                        title="Revoke cluster registration secret failed"
                        variant="danger"
                        isInline
                        component="p"
                    >
                        {getAxiosErrorMessage(error)}
                    </Alert>
                )}
            </Flex>
        </Modal>
    );
}

export default RevokeClusterRegistrationSecretModal;
