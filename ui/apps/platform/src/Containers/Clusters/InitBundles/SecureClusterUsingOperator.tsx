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
                <ExternalLink>
                    <a
                        href={getVersionedDocs(
                            version,
                            'installing/installing_ocp/install-secured-cluster-ocp.html'
                        )}
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        Installing secured cluster services on Red Hat OpenShift
                    </a>
                </ExternalLink>
            )}
            <p>
                You can install secured cluster services on your clusters by using the{' '}
                <strong>SecuredCluster</strong> custom resource.
            </p>
            <Title headingLevel={subHeadingLevel}>
                Procedure to do only once before you secure the first cluster
            </Title>
            <List component="ol">
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
                <ListItem>
                    <p>
                        Download the YAML file for a cluster init bundle. You can use one bundle to
                        secure multiple clusters.
                    </p>
                </ListItem>
                <ListItem>
                    <p>With the ACS project selected create the init bundle secrets in OCP.</p>
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
                                oc -n rhacs-operator -f Operator-secrets-cluster-init-bundle.yaml
                            </ClipboardCopy>
                        </ListItem>
                    </List>
                </ListItem>
            </List>
            <Title headingLevel={subHeadingLevel}>
                Procedure to repeat for each secured cluster
            </Title>
            <p>TODO just use the operator?</p>
        </Flex>
    );
}

export default SecureClusterUsingOperator;
