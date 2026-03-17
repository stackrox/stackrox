import type { ReactElement } from 'react';
import { ClipboardCopy, Flex, List, ListItem, Title } from '@patternfly/react-core';

export type SecureClusterUsingOperatorProps = {
    headingLevel: 'h2' | 'h3';
};

const ocApplyCommand = 'oc create -f <cluster-registration-secret-file>.yaml -n stackrox';
const kubectlApplyCommand = 'kubectl create -f <cluster-registration-secret-file>.yaml -n stackrox';

function SecureClusterUsingOperator({
    headingLevel,
}: SecureClusterUsingOperatorProps): ReactElement {
    const subHeadingLevel = headingLevel === 'h2' ? 'h3' : 'h4';

    return (
        <Flex direction={{ default: 'column' }}>
            <p>
                You can install secured cluster services on your clusters using the{' '}
                <strong>SecuredCluster</strong> custom resource.
            </p>
            <Title headingLevel={subHeadingLevel}>Prerequisites</Title>
            <List component="ul">
                <ListItem>
                    <p>
                        In the RHACS web portal, you have created a cluster registration secret and
                        downloaded its YAML file.
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
                <ListItem>
                    <p>Apply the cluster registration secret on the secured cluster.</p>
                    <p>Apply it using one of the following methods:</p>
                    <List component="ul">
                        <ListItem>
                            <p>
                                On an OpenShift cluster, in the OpenShift Container Platform web
                                console, in the top menu, click <strong>+</strong> to open the{' '}
                                <strong>Import YAML</strong> page.
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
