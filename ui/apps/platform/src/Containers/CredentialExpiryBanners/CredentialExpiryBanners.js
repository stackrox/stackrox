import React from 'react';

import { fetchCentralCertExpiry, fetchScannerCertExpiry } from 'services/CredentialExpiryService';
import {
    generateCentralCertSecret,
    generateScannerCertSecret,
} from 'services/CertGenerationService';
import CredentialExpiry from './CredentialExpiry';

const CredentialExpiryBanners = () => {
    return (
        <>
            <CredentialExpiry
                component="Central"
                expiryFetchFunc={fetchCentralCertExpiry}
                downloadYAMLFunc={generateCentralCertSecret}
            />
            <CredentialExpiry
                component="Scanner"
                expiryFetchFunc={fetchScannerCertExpiry}
                downloadYAMLFunc={generateScannerCertSecret}
            />
        </>
    );
};

export default CredentialExpiryBanners;
