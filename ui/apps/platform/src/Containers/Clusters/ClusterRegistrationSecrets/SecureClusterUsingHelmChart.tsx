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

const codeBlock = [
    'helm install -n stackrox --create-namespace \\',
    'stackrox-secured-cluster-services rhacs/secured-cluster-services \\',
    '--set-file crs.file=<path/to/cluster-registration-secret.yaml> \\',
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
    const subHeadingLevel = headingLevel === 'h2' ? 'h3' : 'h4';
    const [wasCopied, setWasCopied] = useState(false);

    function onClickCopy() {
        // https://developer.mozilla.org/en-US/docs/Web/API/Clipboard/writeText#browser_compatibility
        // Chrome 66 Edge 79 Firefox 63 Safari 13.1
        navigator?.clipboard
            ?.writeText(codeBlock)
            .then(() => {
                setWasCopied(true);
            })
            .catch(() => {
                // TODO addToast(title, message)
            });
    }

    const actions = (
        <CodeBlockAction>
            <ClipboardCopyButton
                aria-label="Copy to clipboard"
                id="ClipboardCopyButton"
                onClick={onClickCopy}
                textId="CodeBlockCode"
                variant="plain"
            >
                {wasCopied ? 'Copied to clipboard' : 'Copy to clipboard'}
            </ClipboardCopyButton>
        </CodeBlockAction>
    );

    return (
        <Flex direction={{ default: 'column' }}>
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
                    <p>You must download the YAML file for a cluster registration secret.</p>
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
