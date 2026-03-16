import type { ReactElement } from 'react';
import { ClipboardCopy, Flex, List, ListItem, Title } from '@patternfly/react-core';

export type SecureClusterUsingOperatorProps = {
    headingLevel: 'h2' | 'h3';
};

function SecureClusterUsingOperator({
    headingLevel,
}: SecureClusterUsingOperatorProps): ReactElement {
    const subHeadingLevel = headingLevel === 'h2' ? 'h3' : 'h4';

    return (
        <Flex direction={{ default: 'column' }}>
            <p>
                You can install secured cluster services on your clusters by using the{' '}
                <strong>SecuredCluster</strong> custom resource.
            </p>
            <Title headingLevel={subHeadingLevel}>Prerequisites</Title>
            <List component="ul">
                <ListItem>
                    <p>
                        In the RHACS web portal, you have created a cluster registration secret and
                        downloaded the YAML file for the cluster registration secret.
                    </p>
                </ListItem>
                <ListItem>
                    <p>
                        In the cluster that you are securing, you have installed the RHACS Operator.
                    </p>
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
                <ListItem>
                    <p>Apply the cluster registration secret on the secured cluster. </p>
                    <p>
                        Perform one of the following tasks to apply the cluster registration
                        secrets:
                    </p>
                    <List component="ul">
                        <ListItem>
                            <p>
                                On an OpenShift cluster: In the OpenShift Container Platform web
                                console on the cluster that you are securing, in the top menu, click{' '}
                                <strong>+</strong> to open the <strong>Import YAML</strong> page.
                            </p>
                            <p>
                                You can drag the cluster registration secret file or copy and paste
                                its contents into the editor, and then click <strong>Create</strong>
                                .
                            </p>
                        </ListItem>
                        <ListItem>
                            <p>
                                On an OpenShift cluster: using the <strong>oc</strong> CLI, run a
                                command similar to the following:
                            </p>
                            <ClipboardCopy>
                                oc create -f &lt;cluster-registration-secret-file&gt;.yaml -n
                                stackrox
                            </ClipboardCopy>
                        </ListItem>
                        <ListItem>
                            <p>
                                On other clusters: using the <strong>kubectl</strong> CLI, run a
                                command similar to the following:
                            </p>
                            <ClipboardCopy>
                                kubectl create -f &lt;cluster-registration-secret-file&gt;.yaml -n
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
