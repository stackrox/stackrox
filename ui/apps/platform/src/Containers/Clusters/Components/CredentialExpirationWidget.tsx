import React from 'react';
import { Link } from 'react-router-dom';
import { Flex, FlexItem } from '@patternfly/react-core';

import useAuthStatus from 'hooks/useAuthStatus';
import { clustersInitBundlesPath } from 'routePaths';

import { ClusterStatus } from '../clusterTypes';
import CredentialExpiration from './CredentialExpiration';
import CredentialInteraction from './CredentialInteraction';

import { isCertificateExpiringSoon } from '../cluster.helpers';

type CredentialExpirationWidgetProps = {
    clusterId: string;
    status: ClusterStatus;
    autoRefreshEnabled: boolean;
    isManagerTypeNonConfigurable: boolean;
};

const CredentialExpirationWidget = ({
    clusterId,
    status,
    autoRefreshEnabled = false,
    isManagerTypeNonConfigurable,
}: CredentialExpirationWidgetProps) => {
    const { currentUser } = useAuthStatus();
    const hasAdminRole = Boolean(currentUser?.userInfo?.roles.some(({ name }) => name === 'Admin')); // optional chaining just in case of the unexpected

    const certExpiryStatus = status?.certExpiryStatus;
    const currentDatetime = new Date();
    // Secured cluster is healthy or has no expiration info => no interaction
    if (
        !certExpiryStatus?.sensorCertExpiry ||
        !isCertificateExpiringSoon(certExpiryStatus, currentDatetime)
    ) {
        return (
            <CredentialExpiration
                certExpiryStatus={certExpiryStatus}
                autoRefreshEnabled={autoRefreshEnabled}
            />
        );
    }
    // Show the link to the token integrations for non-configurable clusters (installed by helm or operator).
    if (isManagerTypeNonConfigurable) {
        return (
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <CredentialExpiration
                        certExpiryStatus={certExpiryStatus}
                        autoRefreshEnabled={autoRefreshEnabled}
                    />
                </FlexItem>
                {hasAdminRole && (
                    <FlexItem>
                        <Link
                            to={clustersInitBundlesPath}
                            className="no-underline flex-shrink-0"
                            data-testid="manageTokens"
                        >
                            Init bundles
                        </Link>
                    </FlexItem>
                )}
            </Flex>
        );
    }
    // Show controls for the certificate renewal
    return (
        <CredentialInteraction
            certExpiryStatus={certExpiryStatus}
            upgradeStatus={status?.upgradeStatus}
            clusterId={clusterId}
            autoRefreshEnabled={autoRefreshEnabled}
        />
    );
};

export default CredentialExpirationWidget;
