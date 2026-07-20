import type { ReactElement } from 'react';

import useCentralCapabilities from 'hooks/useCentralCapabilities';
import useIsLegacyScannerEnabled from 'hooks/useIsLegacyScannerEnabled';
import useIsScannerV4Enabled from 'hooks/useIsScannerV4Enabled';
import usePermissions from 'hooks/usePermissions';

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

    const isLegacyScannerEnabled = useIsLegacyScannerEnabled();
    const isScannerV4Enabled = useIsScannerV4Enabled();

    return (
        <>
            <CredentialExpiryBanner
                component="CENTRAL"
                showCertGenerateAction={showCertGenerateAction}
            />
            <CredentialExpiryBanner
                component="CENTRAL_DB"
                showCertGenerateAction={showCertGenerateAction}
            />
            {isLegacyScannerEnabled && (
                <CredentialExpiryBanner
                    component="SCANNER"
                    showCertGenerateAction={showCertGenerateAction}
                />
            )}
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
