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
    const [errorMessage, setErrorMessage] = useState('');
    const [isRevokingClusterRegistrationSecret, setIsRevokingClusterRegistrationSecret] =
        useState(false);

    function onRevokeClusterRegistrationSecret() {
        setErrorMessage('');
        setIsRevokingClusterRegistrationSecret(true);
        revokeClusterRegistrationSecrets([clusterRegistrationSecret.id])
            .then(({ crsRevocationErrors }) => {
                if (crsRevocationErrors.length === 0) {
                    onCloseModal(true);
                }
                analyticsTrack(REVOKE_CLUSTER_REGISTRATION_SECRET);
            })
            .catch((error) => {
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsRevokingClusterRegistrationSecret(false);
            });
    }

    function onCancel() {
        setErrorMessage('');
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
                    isDisabled={isRevokingClusterRegistrationSecret}
                >
                    Revoke cluster registration secret
                </Button>,
                <Button
                    key="Cancel"
                    variant="secondary"
                    onClick={onCancel}
                    isDisabled={isRevokingClusterRegistrationSecret}
                >
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
                {errorMessage && (
                    <Alert
                        title="Revoke cluster registration secret failed"
                        variant="danger"
                        isInline
                        component="p"
                    >
                        {errorMessage}
                    </Alert>
                )}
            </Flex>
        </Modal>
    );
}

export default RevokeClusterRegistrationSecretModal;
