import React from 'react';
import { Banner } from '@patternfly/react-core';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import { getProductBranding } from 'constants/productBranding';
import useRestQuery from 'hooks/useRestQuery';
import useMetadata from 'hooks/useMetadata';
import { fetchImageIntegrations } from 'services/ImageIntegrationsService';
import { getVersionedDocs } from 'utils/versioning';
import { scannerV4ImageIntegrationType } from 'types/imageIntegration.proto';

function ScannerV4IntegrationBanner() {
    const { version } = useMetadata();
    const { data, isLoading, error } = useRestQuery(fetchImageIntegrations);
    const branding = getProductBranding();

    // Don't show the banner if we don't have successful responses
    if (!data || isLoading || error || !version) {
        return null;
    }

    const hasScannerV4Integration = data.some(({ type }) => type === scannerV4ImageIntegrationType);

    // Don't show the banner if we have a Scanner V4 integration
    if (hasScannerV4Integration) {
        return null;
    }

    const brandedText =
        branding.type === 'RHACS_BRANDING'
            ? 'New Scanner V4 now generally available in RHACS 4.5.'
            : 'New Scanner V4 now generally available in StackRox 4.5.';

    const docsLink = (
        <ExternalLink>
            <a
                href={getVersionedDocs(version, 'operating/examine-images-for-vulnerabilities')}
                target="_blank"
                rel="noopener noreferrer"
            >
                RHACS documentation
            </a>
        </ExternalLink>
    );

    return (
        <Banner variant="blue" className="pf-v5-u-text-align-center">
            {brandedText} For more information, see {docsLink}
        </Banner>
    );
}

export default ScannerV4IntegrationBanner;
