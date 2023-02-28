import { Flex, FlexItem } from '@patternfly/react-core';
import React from 'react';
import { ClusterStatus } from '../clusterTypes';
import CredentialExpiration from './CredentialExpiration';
import CredentialInteraction from './CredentialInteraction';
import ManageTokensButton from './ManageTokensButton';

import { isCertificateExpiringSoon } from '../cluster.helpers';

type CredentialExpirationWidgetProps = {
    clusterId: string;
    status: ClusterStatus;
    isManagerTypeNonConfigurable: boolean;
};

const CredentialExpirationWidget = ({
    clusterId,
    status,
    isManagerTypeNonConfigurable,
}: CredentialExpirationWidgetProps) => {
    const certExpiryStatus = status?.certExpiryStatus;
    const currentDatetime = new Date();
    // Secured cluster is healthy or has no expiration info => no interaction
    if (
        !certExpiryStatus?.sensorCertExpiry ||
        !isCertificateExpiringSoon(certExpiryStatus, currentDatetime)
    ) {
        return <CredentialExpiration certExpiryStatus={certExpiryStatus} />;
    }
    // Show the link to the token integrations for non-configurable clusters (installed by helm or operator).
    if (isManagerTypeNonConfigurable) {
        return (
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <CredentialExpiration certExpiryStatus={certExpiryStatus} />
                </FlexItem>
                <FlexItem>
                    <ManageTokensButton />
                </FlexItem>
            </Flex>
        );
    }
    // Show controls for the certificate renewal
    return (
        <CredentialInteraction
            certExpiryStatus={certExpiryStatus}
            upgradeStatus={status?.upgradeStatus}
            clusterId={clusterId}
        />
    );
};

export default CredentialExpirationWidget;
