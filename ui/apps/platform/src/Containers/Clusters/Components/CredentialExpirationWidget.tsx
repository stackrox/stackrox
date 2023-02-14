import { Flex, FlexItem } from '@patternfly/react-core';
import React from 'react';
import { ClusterStatus } from '../clusterTypes';
import CredentialExpiration from './CredentialExpiration';
import CredentialInteraction from './CredentialInteraction';
import ManageTokensButton from './ManageTokensButton';

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
    if (isManagerTypeNonConfigurable) {
        return (
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <CredentialExpiration certExpiryStatus={status?.certExpiryStatus} />
                </FlexItem>
                <FlexItem>
                    <ManageTokensButton />
                </FlexItem>
            </Flex>
        );
    }
    if (!status?.certExpiryStatus?.sensorCertExpiry) {
        return <CredentialExpiration certExpiryStatus={status?.certExpiryStatus} />;
    }

    return (
        <CredentialInteraction
            certExpiryStatus={status?.certExpiryStatus}
            upgradeStatus={status?.upgradeStatus}
            clusterId={clusterId}
        />
    );
};

export default CredentialExpirationWidget;
