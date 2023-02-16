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
    if (!certExpiryStatus?.sensorCertExpiry) {
        return <CredentialExpiration certExpiryStatus={certExpiryStatus} />;
    }

    if (isManagerTypeNonConfigurable) {
        const currentDatetime = new Date();
        if (!isCertificateExpiringSoon(certExpiryStatus, currentDatetime)) {
            return <CredentialExpiration certExpiryStatus={certExpiryStatus} />;
        }
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

    return (
        <CredentialInteraction
            certExpiryStatus={certExpiryStatus}
            upgradeStatus={status?.upgradeStatus}
            clusterId={clusterId}
        />
    );
};

export default CredentialExpirationWidget;
