import { interactAndWaitForResponses, interceptRequests, waitForResponses } from './request';
import {
    visitSystemConfiguration,
    visitSystemConfigurationWithStaticResponseForPermissions,
} from './systemConfig';

// credentialexpiry

function renderCredentialExpiryBanner(componentUpperCase, expiry, staticResponseForPermissions) {
    const credentialExpiryAlias = 'credentialexpiry';

    const requestConfig = {
        routeMatcherMap: {
            [credentialExpiryAlias]: {
                method: 'GET',
                url: `/v1/credentialexpiry?component=${componentUpperCase}`,
            },
        },
    };

    const staticResponseMap = {
        [credentialExpiryAlias]: {
            body: { expiry },
        },
    };

    interceptRequests(requestConfig, staticResponseMap);

    if (staticResponseForPermissions) {
        visitSystemConfigurationWithStaticResponseForPermissions(staticResponseForPermissions);
    } else {
        visitSystemConfiguration();
    }

    waitForResponses(requestConfig);
}

export function renderCentralCredentialExpiryBanner(expiry, staticResponseForPermissions) {
    renderCredentialExpiryBanner('CENTRAL', expiry, staticResponseForPermissions);
}

export function renderScannerCredentialExpiryBanner(expiry, staticResponseForPermissions) {
    renderCredentialExpiryBanner('SCANNER', expiry, staticResponseForPermissions);
}

// certgen

function interactAndWaitForCertificateDownload(componentLowerCase, interactionCallback) {
    const requestConfig = {
        method: 'POST',
        url: `api/extensions/certgen/${componentLowerCase}`,
    };

    interactAndWaitForResponses(interactionCallback, requestConfig);
}

export function interactAndWaitForCentralCertificateDownload(interactionCallback) {
    interactAndWaitForCertificateDownload('central', interactionCallback);
}

export function interactAndWaitForScannerCertificateDownload(interactionCallback) {
    interactAndWaitForCertificateDownload('scanner', interactionCallback);
}
