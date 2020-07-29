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
                component="StackRox Central"
                expiryFetchFunc={fetchCentralCertExpiry}
                downloadYAMLFunc={generateCentralCertSecret}
            />
            <CredentialExpiry
                component="StackRox Scanner"
                expiryFetchFunc={fetchScannerCertExpiry}
                downloadYAMLFunc={generateScannerCertSecret}
            />
        </>
    );
};

export default CredentialExpiryBanners;
