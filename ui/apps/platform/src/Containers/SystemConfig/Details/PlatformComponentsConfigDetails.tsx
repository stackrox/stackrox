import React, { ReactElement } from 'react';
import {
    Button,
    Card,
    CardBody,
    CardTitle,
    CodeBlock,
    Divider,
    Grid,
    GridItem,
    Stack,
    StackItem,
    Text,
} from '@patternfly/react-core';

import './PlatformComponentsConfigDetails.css';

// @TODO: Add platformComponentsConfig as prop
const PlatformComponentsConfigDetails = (): ReactElement => {
    return (
        <Grid hasGutter md={3}>
            <GridItem span={4}>
                <Card isFlat>
                    <CardTitle>Core system</CardTitle>
                    <CardBody>
                        <Stack hasGutter>
                            <Text>
                                Components found in core Openshift and Kubernetes namespaces are
                                included in the platform definition by default.
                            </Text>
                            <Divider component="div" />
                            <Text component="small" className="pf-v5-u-color-200">
                                Namespaces match (Regex)
                            </Text>
                            <CodeBlock>^kube-.*|^openshift-.*</CodeBlock>
                        </Stack>
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem span={4}>
                <Card isFlat>
                    <CardTitle>Red Hat layered products</CardTitle>
                    <CardBody>
                        <Stack hasGutter>
                            <Text>
                                Components found in Red Hat layered and partner product namespaces
                                are included in the platform definition by default.
                            </Text>
                            <Divider component="div" />
                            <Text component="small" className="pf-v5-u-color-200">
                                Namespaces match (Regex)
                            </Text>
                            <CodeBlock>
                                <div className="truncate-multiline">
                                    ^stackrox$|^rhacs-operator$|^open-cluster-management$|^multicluster-engine$|^aap$|^hive$`
                                </div>
                                <Button variant="link" isInline className="pf-v5-u-mt-sm">
                                    Show more
                                </Button>
                            </CodeBlock>
                        </Stack>
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem span={4}>
                <Card isFlat>
                    <CardTitle>Custom components</CardTitle>
                    <CardBody>
                        <Stack hasGutter>
                            <Text>
                                Extend the platform definition by defining namespaces for additional
                                applications and products.
                            </Text>
                            <Divider component="div" />
                            <Button variant="link" isInline className="pf-v5-u-mt-sm">
                                Show 3 custom components
                            </Button>
                        </Stack>
                    </CardBody>
                </Card>
            </GridItem>
        </Grid>
    );
};

export default PlatformComponentsConfigDetails;
