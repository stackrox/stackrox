import type { ReactElement, ReactNode } from 'react';
import { Alert, Content } from '@patternfly/react-core';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useMetadata from 'hooks/useMetadata';
import { getVersionedDocs } from 'utils/versioning';

export type InstallMethodDeprecationAlertProps = {
    deprecationMessage: ReactNode;
};

/**
 * Shared deprecation warning alert used across cluster installation pages.
 * Displays a configurable deprecation message and a link to operator installation docs.
 */
function InstallMethodDeprecationAlert({
    deprecationMessage,
}: InstallMethodDeprecationAlertProps): ReactElement {
    const { version } = useMetadata();

    return (
        <Alert
            title="Deprecation notice"
            component="p"
            variant="warning"
            isInline
            className="pf-v6-u-mb-lg"
        >
            <Content component="p">{deprecationMessage}</Content>
            <Content component="p">
                Use the Kubernetes operator to install secured cluster services instead.
                {version && (
                    <>
                        {' '}
                        See{' '}
                        <ExternalLink>
                            <a
                                href={getVersionedDocs(
                                    version,
                                    'installing/installing-rhacs-on-red-hat-openshift#install-secured-cluster-ocp'
                                )}
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                operator installation documentation
                            </a>
                        </ExternalLink>
                        .
                    </>
                )}
            </Content>
        </Alert>
    );
}

export default InstallMethodDeprecationAlert;
