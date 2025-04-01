import React, { ReactElement } from 'react';

import useCentralCapabilities from 'hooks/useCentralCapabilities';
import useIsScannerV4Enabled from 'hooks/useIsScannerV4Enabled';
import usePermissions from 'hooks/usePermissions';

import AnnouncementBanner from './AnnouncementBanner';
import CredentialExpiryBanner from './CredentialExpiryBanner';
import DatabaseStatusBanner from './DatabaseStatusBanner';
import OutdatedVersionBanner from './OutdatedVersionBanner';
import ServerStatusBanner from './ServerStatusBanner';

function Banners(): ReactElement {
    // Assume MainPage renders this element only after feature flags and permissions are available.
    const { hasReadWriteAccess } = usePermissions();

    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const centralCanUpdateCert = isCentralCapabilityAvailable('centralCanUpdateCert');
    const hasAdministrationWritePermission = hasReadWriteAccess('Administration');
    const showCertGenerateAction = centralCanUpdateCert && hasAdministrationWritePermission;

    const isScannerV4Enabled = useIsScannerV4Enabled();

    return (
        <>
            <AnnouncementBanner />
            <CredentialExpiryBanner
                component="CENTRAL"
                showCertGenerateAction={showCertGenerateAction}
            />
            <CredentialExpiryBanner
                component="CENTRAL_DB"
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
