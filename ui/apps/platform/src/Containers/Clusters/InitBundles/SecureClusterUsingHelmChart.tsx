import React, { ReactElement, useState } from 'react';
import {
    ClipboardCopy,
    ClipboardCopyButton,
    CodeBlock,
    CodeBlockAction,
    CodeBlockCode,
    Flex,
    List,
    ListItem,
    Title,
} from '@patternfly/react-core';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useMetadata from 'hooks/useMetadata';
import { getVersionedDocs } from 'utils/versioning';
import useClipboardCopy from 'hooks/useClipboardCopy';

const codeBlock = [
    'helm install -n stackrox --create-namespace \\',
    'stackrox-secured-cluster-services rhacs/secured-cluster-services \\',
    '-f <path/to/Helm-values-cluster-init-bundle.yaml> \\',
    '--set clusterName=<name_of_the_secured_cluster> \\',
    '--set centralEndpoint=<endpoint_of_central_service> \\',
    '--set imagePullSecrets.username=<your redhat.com developer account username> \\',
    '--set imagePullSecrets.password=<your redhat.com developer account password>',
].join('\n');

export type SecureClusterUsingHelmChartProps = {
    headingLevel: 'h2' | 'h3';
};

function SecureClusterUsingHelmChart({
    headingLevel,
}: SecureClusterUsingHelmChartProps): ReactElement {
    const { version } = useMetadata();
    const subHeadingLevel = headingLevel === 'h2' ? 'h3' : 'h4';
    const { wasCopied, copyToClipboard } = useClipboardCopy();

    const actions = (
        <CodeBlockAction>
            <ClipboardCopyButton
                aria-label="Copy to clipboard"
                id="ClipboardCopyButton"
                onClick={() => copyToClipboard(codeBlock)}
                textId="CodeBlockCode"
                variant="plain"
            >
                {wasCopied ? 'Copied to clipboard' : 'Copy to clipboard'}
            </ClipboardCopyButton>
        </CodeBlockAction>
    );

    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel={headingLevel}>
                Secure a cluster using Helm chart installation method
            </Title>
            {version && (
                <>
                    <ExternalLink>
                        <a
                            href={getVersionedDocs(
                                version,
                                'installing/installing-rhacs-on-other-platforms#init-bundle-other'
                            )}
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            Generating and applying an init bundle for RHACS on other platforms
                        </a>
                    </ExternalLink>
                    <ExternalLink>
                        <a
                            href={getVersionedDocs(
                                version,
                                'installing/installing-rhacs-on-other-platforms#install-secured-cluster-other'
                            )}
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            Installing secured cluster services for RHACS on other platforms
                        </a>
                    </ExternalLink>
                </>
            )}
            <Title headingLevel={subHeadingLevel}>Prerequisites</Title>
            <List component="ul">
                <ListItem>
                    <p>
                        You must have previously added the Helm chart repository, or add it using
                        the following command:
                    </p>
                    <ClipboardCopy>
                        helm repo add rhacs https://mirror.openshift.com/pub/rhacs/charts/
                    </ClipboardCopy>
                </ListItem>
                <ListItem>
                    <p>
                        You must download the YAML file for a cluster init bundle. You can use one
                        bundle to secure multiple clusters.
                    </p>
                </ListItem>
                <ListItem>
                    <p>
                        You must have access to the Red Hat Container Registry and a pull secret for
                        authentication.
                    </p>
                </ListItem>
                <ListItem>
                    <p>
                        You must have the address and the port number that you are exposing the
                        Central service on.
                    </p>
                </ListItem>
            </List>
            <Title headingLevel={subHeadingLevel}>Repeat for each secured cluster</Title>
            <List component="ol">
                <ListItem>
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <p>Run a command similar to the following:</p>
                        <CodeBlock actions={actions}>
                            <CodeBlockCode>{codeBlock}</CodeBlockCode>
                        </CodeBlock>
                    </Flex>
                </ListItem>
            </List>
        </Flex>
    );
}

export default SecureClusterUsingHelmChart;
