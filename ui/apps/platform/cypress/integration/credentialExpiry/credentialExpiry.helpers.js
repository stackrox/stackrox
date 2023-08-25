import {
    interactAndWaitForResponses,
    interceptRequests,
    waitForResponses,
} from '../../helpers/request';

import {
    visitSystemConfiguration,
    visitSystemConfigurationWithStaticResponseForPermissions,
} from '../systemConfig/systemConfig.helpers';

// credentialexpiry

function visitSystemConfigurationWithCredentialExpiryBanner(
    componentUpperCase,
    expiry,
    staticResponseForPermissions
) {
    const credentialExpiryAlias = 'credentialexpiry';

    const routeMatcherMap = {
        [credentialExpiryAlias]: {
            method: 'GET',
            url: `/v1/credentialexpiry?component=${componentUpperCase}`,
        },
    };

    const staticResponseMap = {
        [credentialExpiryAlias]: {
            body: { expiry },
        },
    };

    interceptRequests(routeMatcherMap, staticResponseMap);

    if (staticResponseForPermissions) {
        visitSystemConfigurationWithStaticResponseForPermissions(staticResponseForPermissions);
    } else {
        visitSystemConfiguration();
    }

    waitForResponses(routeMatcherMap);
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
    const certgenAlias = 'certgen';

    const routeMatcherMap = {
        [certgenAlias]: {
            method: 'POST',
            url: `api/extensions/certgen/${componentLowerCase}`,
        },
    };

    interactAndWaitForResponses(interactionCallback, routeMatcherMap);
}

export function interactAndWaitForCentralCertificateDownload(interactionCallback) {
    interactAndWaitForCertificateDownload('central', interactionCallback);
}

export function interactAndWaitForScannerCertificateDownload(interactionCallback) {
    interactAndWaitForCertificateDownload('scanner', interactionCallback);
}
