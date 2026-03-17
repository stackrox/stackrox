import type { ReactElement } from 'react';
import { ClipboardCopy, Flex, List, ListItem, Title } from '@patternfly/react-core';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useMetadata from 'hooks/useMetadata';
import { getVersionedDocs } from 'utils/versioning';

export type SecureClusterUsingOperatorProps = {
    headingLevel: 'h2' | 'h3';
};

const ocApplyCommand = 'oc create -f <init-bundle-file>.yaml -n stackrox';
const kubectlApplyCommand = 'kubectl create -f <init-bundle-file>.yaml -n stackrox';

function SecureClusterUsingOperator({
    headingLevel,
}: SecureClusterUsingOperatorProps): ReactElement {
    const { version } = useMetadata();
    const subHeadingLevel = headingLevel === 'h2' ? 'h3' : 'h4';

    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel={headingLevel}>
                Secure a cluster using the Operator installation method
            </Title>
            {version && (
                <>
                    <ExternalLink>
                        <a
                            href={getVersionedDocs(
                                version,
                                'installing/installing-rhacs-on-red-hat-openshift#init-bundle-ocp'
                            )}
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            Generating and applying an init bundle for RHACS on Red Hat OpenShift
                            (OpenShift)
                        </a>
                    </ExternalLink>
                    <ExternalLink>
                        <a
                            href={getVersionedDocs(
                                version,
                                'installing/installing-rhacs-on-red-hat-openshift#install-secured-cluster-ocp'
                            )}
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            Installing RHACS on secured clusters by using the Operator (OpenShift)
                        </a>
                    </ExternalLink>
                    {/* TODO ROX-33550: Add non-OpenShift operator documentation links when available */}
                </>
            )}
            <p>
                You can install secured cluster services on your clusters using the{' '}
                <strong>SecuredCluster</strong> custom resource.
            </p>
            <Title headingLevel={subHeadingLevel}>Prerequisites</Title>
            <List component="ul">
                <ListItem>
                    <p>
                        In the RHACS web portal, you have created an init bundle and downloaded its
                        YAML file.
                    </p>
                </ListItem>
                <ListItem>
                    <p>You have installed the RHACS Operator on the cluster you are securing.</p>
                    <p>
                        For Operator installation, create a new project or namespace.{' '}
                        <strong>rhacs-operator</strong> is a good name choice.
                    </p>
                    <p>To install the RHACS Operator:</p>
                    <List component="ul">
                        <ListItem>
                            On Red Hat OpenShift Container Platform, use{' '}
                            <strong>Operators &gt; OperatorHub</strong> in the web console.
                        </ListItem>
                        <ListItem>
                            On other platforms, apply an image pull secret and use the{' '}
                            <strong>rhacs-operator</strong> Helm chart.
                        </ListItem>
                    </List>
                </ListItem>
            </List>
            <Title headingLevel={subHeadingLevel}>Repeat for each secured cluster</Title>
            <List component="ol">
                <ListItem>
                    <p>Apply the init bundle on the secured cluster.</p>
                    <p>
                        Applying the init bundle creates the secrets and resources that the secured
                        cluster needs to communicate with RHACS. Apply it using one of the following
                        methods:
                    </p>
                    {version && (
                        <ExternalLink>
                            <a
                                href={getVersionedDocs(
                                    version,
                                    'rhacs_cloud_service/setting-up-rhacs-cloud-service-with-red-hat-openshift-secured-clusters#init-bundle-cloud-ocp-apply'
                                )}
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                Creating resources by using the init bundle
                            </a>
                        </ExternalLink>
                    )}
                    <List component="ul">
                        <ListItem>
                            <p>
                                On an OpenShift cluster, in the OpenShift Container Platform web
                                console, in the top menu, click <strong>+</strong> to open the{' '}
                                <strong>Import YAML</strong> page.
                            </p>
                            <p>
                                You can drag the init bundle file or copy and paste its contents
                                into the editor, and then click <strong>Create</strong>.
                            </p>
                        </ListItem>
                        <ListItem>
                            <p>
                                On an OpenShift cluster: using the <strong>oc</strong> CLI, run a
                                command similar to the following:
                            </p>
                            <ClipboardCopy>{ocApplyCommand}</ClipboardCopy>
                        </ListItem>
                        <ListItem>
                            <p>
                                On other clusters: using the <strong>kubectl</strong> CLI, run a
                                command similar to the following:
                            </p>
                            <ClipboardCopy>{kubectlApplyCommand}</ClipboardCopy>
                        </ListItem>
                    </List>
                </ListItem>
                <ListItem>
                    <p>Install secured cluster services on the cluster using the RHACS Operator.</p>
                </ListItem>
            </List>
        </Flex>
    );
}

export default SecureClusterUsingOperator;
