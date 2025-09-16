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
                                'installing/installing-rhacs-on-red-hat-openshift#init-bundle-ocp'
                            )}
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            Generating and applying an init bundle for RHACS on Red Hat OpenShift
                        </a>
                    </ExternalLink>
                    <ExternalLink>
                        <a
                            href={getVersionedDocs(
                                version,
                                'installing/installing-rhacs-on-red-hat-openshift#install-secured-cluster-operator_install-secured-cluster-ocp'
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
                        In the RHACS web portal, you have created an init bundle and downloaded the
                        YAML file for the init bundle.
                    </p>
                </ListItem>
                <ListItem>
                    <p>
                        In the Red Hat OpenShift Container Platform web console on the cluster that
                        you are securing, you have installed the RHACS Operator.
                    </p>
                    <p>
                        For Operator installation, create a new Red Hat OpenShift Container Platform
                        project. <strong>rhacs-operator</strong> is a good name choice.
                    </p>
                </ListItem>
            </List>
            <Title headingLevel={subHeadingLevel}>Repeat for each secured cluster</Title>
            <List component="ol">
                <ListItem>
                    <p>Apply the init bundle on the secured cluster. </p>
                    <p>
                        Applying the init bundle creates the secrets and resources that the secured
                        cluster needs to communicate with RHACS. Perform one of the following tasks
                        to apply the init bundle:
                    </p>
                    {version && (
                        <ExternalLink>
                            <a
                                href={getVersionedDocs(
                                    version,
                                    'rhacs_cloud_service/setting-up-rhacs-cloud-service-with-red-hat-openshift-secured-clusters#create-resource-init-bundle_init-bundle-cloud-ocp-apply'
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
                                In the OpenShift Container Platform web console on the cluster that
                                you are securing, in the top menu, click <strong>+</strong> to open
                                the <strong>Import YAML</strong> page.
                            </p>
                            <p>
                                You can drag the init bundle file or copy and paste its contents
                                into the editor, and then click <strong>Create</strong>.
                            </p>
                        </ListItem>
                        <ListItem>
                            <p>
                                On the cluster that you are securing, using the Red Hat OpenShift
                                CLI, run a command similar to the following:
                            </p>
                            <ClipboardCopy>
                                oc create -f name-Operator-secrets-cluster-init-bundle.yaml -n
                                stackrox
                            </ClipboardCopy>
                        </ListItem>
                    </List>
                </ListItem>
                <ListItem>
                    <p>
                        On the cluster that you are securing, install secured cluster services using
                        the RHACS Operator.
                    </p>
                </ListItem>
            </List>
        </Flex>
    );
}

export default SecureClusterUsingOperator;
