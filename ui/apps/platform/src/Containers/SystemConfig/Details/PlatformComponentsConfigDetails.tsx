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

import { PlatformComponentRule, PlatformComponentsConfig } from 'types/config.proto';

import './PlatformComponentsConfigDetails.css';

// @TODO: Remove hardcoded value and add platformComponentsConfig as a prop
const platformComponentsConfig: PlatformComponentsConfig = {
    needsReevaluation: false,
    rules: [
        {
            name: 'system rule',
            namespaceRule: {
                regex: '^kube-.*|^openshift-.*',
            },
        },
        {
            name: 'red hat layered products',
            namespaceRule: {
                regex: '^stackrox$|^rhacs-operator$|^open-cluster-management$|^multicluster-engine$|^aap$|^hive$`',
            },
        },
        {
            name: 'custom platform component 1',
            namespaceRule: {
                regex: '^my-application$|^custom-test$|^something-else$',
            },
        },
        {
            name: 'custom platform component 2',
            namespaceRule: {
                regex: '^nvidia$',
            },
        },
    ],
};

const PlatformComponentsConfigDetails = (): ReactElement => {
    let coreSystemRule: PlatformComponentRule | undefined;
    let redHatLayeredProductsRule: PlatformComponentRule | undefined;
    const customRules: PlatformComponentRule[] = [];

    platformComponentsConfig.rules.forEach((rule) => {
        if (rule.name === 'system rule') {
            coreSystemRule = rule;
        } else if (rule.name === 'red hat layered products') {
            redHatLayeredProductsRule = rule;
        } else {
            customRules.push(rule);
        }
    });

    return (
        <Grid hasGutter>
            <GridItem sm={12} md={6} lg={4}>
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
                            <CodeBlock>{coreSystemRule?.namespaceRule.regex}</CodeBlock>
                        </Stack>
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem sm={12} md={6} lg={4}>
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
                                    {redHatLayeredProductsRule?.namespaceRule.regex}
                                </div>
                            </CodeBlock>
                            <StackItem className="pf-v5-u-text-align-center pf-v5-u-mt-sm">
                                <Button variant="link" isInline>
                                    View more
                                </Button>
                            </StackItem>
                        </Stack>
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem sm={12} md={6} lg={4}>
                <Card isFlat>
                    <CardTitle>Custom components</CardTitle>
                    <CardBody>
                        <Stack hasGutter>
                            <Text>
                                Extend the platform definition by defining namespaces for additional
                                applications and products.
                            </Text>
                            <Divider component="div" />
                            <Text component="small" className="pf-v5-u-color-200">
                                Namespaces match (Regex)
                            </Text>
                            {customRules.length === 0 && <CodeBlock>None</CodeBlock>}
                            {customRules.length >= 1 && (
                                <CodeBlock>
                                    <Text component="small" className="pf-v5-u-color-200">
                                        {customRules[0].name}
                                    </Text>
                                    <div className="truncate-multiline">
                                        {customRules[0].namespaceRule.regex}
                                    </div>
                                </CodeBlock>
                            )}
                            {customRules.length > 1 && (
                                <StackItem className="pf-v5-u-text-align-center pf-v5-u-mt-sm">
                                    <Button variant="link" isInline>
                                        View more
                                    </Button>
                                </StackItem>
                            )}
                        </Stack>
                    </CardBody>
                </Card>
            </GridItem>
        </Grid>
    );
};

export default PlatformComponentsConfigDetails;
