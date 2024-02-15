import React from 'react';
import { Banner, Flex, FlexItem } from '@patternfly/react-core';
import { InfoCircleIcon } from '@patternfly/react-icons';

import useRestQuery from 'hooks/useRestQuery';
import useMetadata from 'hooks/useMetadata';
import { fetchImageIntegrations } from 'services/ImageIntegrationsService';
import { getVersionedDocs } from 'utils/versioning';
import { scannerV4ImageIntegrationType } from 'types/imageIntegration.proto';

function ScannerV4IntegrationBanner() {
    const { version } = useMetadata();
    const { data, loading, error } = useRestQuery(fetchImageIntegrations);

    // Don't show the banner if we don't have successful responses
    if (!data || loading || error || !version) {
        return null;
    }

    const hasScannerV4Integration = data.some(({ type }) => type === scannerV4ImageIntegrationType);

    // Don't show the banner if we have a Scanner V4 integration
    if (hasScannerV4Integration) {
        return null;
    }

    return (
        <Banner variant="info">
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                justifyContent={{ default: 'justifyContentCenter' }}
            >
                <FlexItem>
                    <InfoCircleIcon />
                </FlexItem>
                <FlexItem>
                    New Scanner V4 now generally available in RHACS 4.4. Refer to the{' '}
                    {/* TODO When we have a more specific link to Scanner v4 we may want to change this URL */}
                    <a
                        href={getVersionedDocs(
                            version,
                            'integration/integrate-with-image-vulnerability-scanners.html'
                        )}
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        RHACS documentation
                    </a>{' '}
                    to learn more.
                </FlexItem>
            </Flex>
        </Banner>
    );
}

export default ScannerV4IntegrationBanner;
