import React, { ReactElement } from 'react';
import { ClipboardCopy, Flex, List, ListItem, Title } from '@patternfly/react-core';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useMetadata from 'hooks/useMetadata';
import { getVersionedDocs } from 'utils/versioning';

export type SecureClusterUsingOperatorProps = {
    headingLevel: 'h2' | 'h3';
};

function SecureClusterUsingOperator({
    headingLevel,
}: SecureClusterUsingOperatorProps): ReactElement {
    const { version } = useMetadata();
    const subHeadingLevel = headingLevel === 'h2' ? 'h3' : 'h4';

    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel={headingLevel}>
                Secure a cluster using Operator installation method
            </Title>
            {version && (
                <>
                    <ExternalLink>
                        <a
                            href={getVersionedDocs(
                                version,
                                'installing/installing_ocp/init-bundle-ocp.html'
                            )}
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            Generating and applying an init bundle for RHACS on Red HatOpenShift
                        </a>
                    </ExternalLink>
                    <ExternalLink>
                        <a
                            href={getVersionedDocs(
                                version,
                                'installing/installing_ocp/install-secured-cluster-ocp.html#installing-sc-operator'
                            )}
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            Installing RHACS on secured clusters by using the Operator
                        </a>
                    </ExternalLink>
                </>
            )}
            <p>
                You can install secured cluster services on your clusters by using the{' '}
                <strong>SecuredCluster</strong> custom resource.
            </p>
            <Title headingLevel={subHeadingLevel}>Prerequisites</Title>
            <List component="ul">
                <ListItem>
                    <p>
                        Download the YAML file for a cluster init bundle. You can use one bundle to
                        secure multiple clusters.
                    </p>
                </ListItem>
                <ListItem>
                    <p>
                        Use the Red Hat OpenShift Container Platform web console to install the
                        RHACS Operator from OperatorHub.
                    </p>
                    <p>
                        Create a new Red Hat OpenShift Container Platform project for RHACS.{' '}
                        <strong>rhacs-operator</strong> is a good name choice.
                    </p>
                </ListItem>
            </List>
            <Title headingLevel={subHeadingLevel}>Repeat for each secured cluster</Title>
            <List component="ol">
                <ListItem>
                    <p>Apply the init bundle by creating a resource on the secured cluster.</p>
                    <p>
                        With the RHACS project selected create the init bundle secrets in OpenShift
                        Container Platform.
                    </p>
                    {version && (
                        <ExternalLink>
                            <a
                                href={getVersionedDocs(
                                    version,
                                    'cloud_service/installing_cloud_ocp/init-bundle-cloud-ocp-apply.html#create-resource-init-bundle_init-bundle-cloud-ocp-apply'
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
                                In the OpenShift Container Platform web console, in the top menu,
                                click <strong>+</strong> to open the <strong>Import YAML</strong>{' '}
                                page.
                            </p>
                            <p>
                                You can drag the init bundle file or copy and paste its contents
                                into the editor, and then click <strong>Create</strong>.
                            </p>
                        </ListItem>
                        <ListItem>
                            <p>
                                Using the Red Hat OpenShift CLI, run a command similar to the
                                following:
                            </p>
                            <ClipboardCopy>
                                oc create -n rhacs-operator -f
                                Operator-secrets-cluster-init-bundle.yaml
                            </ClipboardCopy>
                        </ListItem>
                    </List>
                </ListItem>
                <ListItem>
                    <p>
                        Install secured cluster services on each cluster using the RHACS operator.
                    </p>
                </ListItem>
            </List>
        </Flex>
    );
}

export default SecureClusterUsingOperator;
