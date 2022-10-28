import { interactAndWaitForResponses, interceptRequests, waitForResponses } from './request';
import {
    visitSystemConfiguration,
    visitSystemConfigurationWithStaticResponseForPermissions,
} from './systemConfig';

// credentialexpiry

function visitSystemConfigurationWithCredentialExpiryBanner(
    componentUpperCase,
    expiry,
    staticResponseForPermissions
) {
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

export function visitSystemConfigurationWithCentralCredentialExpiryBanner(
    expiry,
    staticResponseForPermissions
) {
    visitSystemConfigurationWithCredentialExpiryBanner(
        'CENTRAL',
        expiry,
        staticResponseForPermissions
    );
}

export function visitSystemConfigurationWithScannerCredentialExpiryBanner(
    expiry,
    staticResponseForPermissions
) {
    visitSystemConfigurationWithCredentialExpiryBanner(
        'SCANNER',
        expiry,
        staticResponseForPermissions
    );
}

// certgen

function interactAndWaitForCertificateDownload(componentLowerCase, interactionCallback) {
    const requestConfig = {
        reouteMatcherMap: {
            method: 'POST',
            url: `api/extensions/certgen/${componentLowerCase}`,
        },
    };

    interactAndWaitForResponses(interactionCallback, requestConfig);
}

export function interactAndWaitForCentralCertificateDownload(interactionCallback) {
    interactAndWaitForCertificateDownload('central', interactionCallback);
}

export function interactAndWaitForScannerCertificateDownload(interactionCallback) {
    interactAndWaitForCertificateDownload('scanner', interactionCallback);
}
