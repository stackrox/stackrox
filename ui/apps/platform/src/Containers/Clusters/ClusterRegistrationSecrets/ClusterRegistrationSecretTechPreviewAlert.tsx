import React from 'react';
import { Alert } from '@patternfly/react-core';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import { getVersionedDocs } from 'utils/versioning';
import useMetadata from 'hooks/useMetadata';

export default function ClusterRegistrationSecretTechPreviewAlert() {
    const { version } = useMetadata();
    return (
        <Alert
            isInline
            component="p"
            variant="info"
            title="Cluster registration secrets are a Technology Preview feature"
        >
            Cluster registration secrets (or &ldquo;CRS&rdquo; for short) are a modern alternative
            to init bundles and will at some point replace init bundles entirely. Cluster
            registration secrets and init bundles differ in their specific usage semantics &mdash;
            please consult the{' '}
            <ExternalLink>
                <a
                    href={getVersionedDocs(
                        version,
                        'installing/installing-rhacs-on-red-hat-openshift#init-bundle-ocp'
                    )}
                    rel="noopener noreferrer"
                    target="_blank"
                >
                    RHACS documentation
                </a>
            </ExternalLink>{' '}
            for details. In any case, you only need an init bundle or a CRS to secure a new cluster,
            not both.
        </Alert>
    );
}
