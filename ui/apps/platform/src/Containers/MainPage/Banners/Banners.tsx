import React, { ReactElement } from 'react';

import useCentralCapabilities from 'hooks/useCentralCapabilities';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';

import AnnouncementBanner from './AnnouncementBanner';
import CredentialExpiryBanner from './CredentialExpiryBanner';
import DatabaseStatusBanner from './DatabaseStatusBanner';
import OutdatedVersionBanner from './OutdatedVersionBanner';
import ServerStatusBanner from './ServerStatusBanner';

function Banners(): ReactElement {
    // Assume MainPage renders this element only after feature flags and permissions are available.
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { hasReadWriteAccess } = usePermissions();

    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const centralCanUpdateCert = isCentralCapabilityAvailable('centralCanUpdateCert');
    const hasAdministrationWritePermission = hasReadWriteAccess('Administration');
    const showCertGenerateAction = centralCanUpdateCert && hasAdministrationWritePermission;

    const isScannerV4Enabled = isFeatureFlagEnabled('ROX_SCANNER_V4');

    return (
        <>
            <AnnouncementBanner />
            <CredentialExpiryBanner
                component="CENTRAL"
                showCertGenerateAction={showCertGenerateAction}
            />
            <CredentialExpiryBanner
                component="SCANNER"
                showCertGenerateAction={showCertGenerateAction}
            />
            {isScannerV4Enabled && (
                <CredentialExpiryBanner
                    component="SCANNER_V4"
                    showCertGenerateAction={showCertGenerateAction}
                />
            )}
            <OutdatedVersionBanner />
            <DatabaseStatusBanner />
            <ServerStatusBanner />
        </>
    );
}

export default Banners;
