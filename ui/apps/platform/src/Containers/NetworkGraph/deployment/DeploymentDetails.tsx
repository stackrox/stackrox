import React, { useState } from 'react';
import {
    Button,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    ExpandableSection,
    Flex,
    FlexItem,
    Label,
    LabelGroup,
    Stack,
    StackItem,
    Text,
    TextContent,
    TextVariants,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

function DetailSection({ title, children }) {
    const [isExpanded, setIsExpanded] = useState(true);

    const onToggle = (_isExpanded: boolean) => {
        setIsExpanded(_isExpanded);
    };

    return (
        <ExpandableSection
            isExpanded={isExpanded}
            onToggle={onToggle}
            toggleContent={
                <TextContent>
                    <Text component={TextVariants.h1} className="pf-u-font-size-xl">
                        {title}
                    </Text>
                </TextContent>
            }
        >
            <div className="pf-u-px-sm pf-u-pb-md">{children}</div>
        </ExpandableSection>
    );
}

function DeploymentDetails() {
    return (
        <div className="pf-u-h-100 pf-u-p-md">
            <ul>
                <li>
                    <DetailSection title="Security overview">
                        <DescriptionList columnModifier={{ default: '2Col' }}>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Risk score</DescriptionListTerm>
                                <DescriptionListDescription>Priority 1</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Asset value</DescriptionListTerm>
                                <DescriptionListDescription>High</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Violations</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <Flex
                                        direction={{ default: 'row' }}
                                        alignItems={{ default: 'alignItemsCenter' }}
                                    >
                                        <FlexItem>
                                            <ExclamationCircleIcon className="pf-u-danger-color-100" />
                                        </FlexItem>
                                        <FlexItem>
                                            <Button variant="link" isInline>
                                                1 deploy
                                            </Button>
                                            ,{' '}
                                            <Button variant="link" isInline>
                                                1 runtime
                                            </Button>
                                        </FlexItem>
                                    </Flex>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Processes</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <Flex
                                        direction={{ default: 'row' }}
                                        alignItems={{ default: 'alignItemsCenter' }}
                                    >
                                        <FlexItem>
                                            <ExclamationCircleIcon className="pf-u-danger-color-100" />
                                        </FlexItem>
                                        <FlexItem>
                                            <Button variant="link" isInline>
                                                3 anomalous
                                            </Button>
                                            ,{' '}
                                            <Button variant="link" isInline>
                                                12 running
                                            </Button>
                                        </FlexItem>
                                    </Flex>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </DetailSection>
                </li>
                <Divider component="li" className="pf-u-mb-sm" />
                <li>
                    <DetailSection title="Network security">
                        <DescriptionList columnModifier={{ default: '1Col' }}>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Network policy rules</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <Flex
                                        direction={{ default: 'row' }}
                                        alignItems={{ default: 'alignItemsCenter' }}
                                    >
                                        <FlexItem>
                                            <ExclamationCircleIcon className="pf-u-warning-color-100" />
                                        </FlexItem>
                                        <FlexItem>
                                            0 egress,{' '}
                                            <Button variant="link" isInline>
                                                1 ingress
                                            </Button>
                                        </FlexItem>
                                    </Flex>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Flows observed</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <Flex
                                        direction={{ default: 'row' }}
                                        alignItems={{ default: 'alignItemsCenter' }}
                                    >
                                        <FlexItem>
                                            <ExclamationCircleIcon className="pf-u-danger-color-100" />
                                        </FlexItem>
                                        <FlexItem>
                                            <Button variant="link" isInline>
                                                3 external
                                            </Button>
                                            ,{' '}
                                            <Button variant="link" isInline>
                                                2 anomalous
                                            </Button>
                                            ,{' '}
                                            <Button variant="link" isInline>
                                                4 active
                                            </Button>
                                            ,{' '}
                                            <Button variant="link" isInline>
                                                312 allowed
                                            </Button>
                                        </FlexItem>
                                    </Flex>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </DetailSection>
                </li>
                <Divider component="li" className="pf-u-mb-sm" />
                <li>
                    <DetailSection title="Deployment configuration">
                        <Stack hasGutter>
                            <StackItem>
                                <DescriptionList columnModifier={{ default: '2Col' }}>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Name</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <Button variant="link" isInline>
                                                visa-processor
                                            </Button>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Cluster</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <Button variant="link" isInline>
                                                Production
                                            </Button>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Created</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            12/09/21 | 6:03:23 PM
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Namespace</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <Button variant="link" isInline>
                                                Naples
                                            </Button>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Replicas</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <Button variant="link" isInline>
                                                2 pods
                                            </Button>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Service account</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <Button variant="link" isInline>
                                                visa-processor
                                            </Button>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                </DescriptionList>
                            </StackItem>
                            <StackItem>
                                <DescriptionList columnModifier={{ default: '1Col' }}>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Labels</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <LabelGroup>
                                                <Label color="blue">app:visa-processor</Label>
                                                <Label color="blue">
                                                    helm.sh/release-namespace:naples
                                                </Label>
                                            </LabelGroup>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Annotations</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <LabelGroup>
                                                <Label color="blue">
                                                    deprecated.daemonset.template.generation:15
                                                </Label>
                                                <Label color="blue">
                                                    email:support@stackrox.com
                                                </Label>
                                            </LabelGroup>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                </DescriptionList>
                            </StackItem>
                            <StackItem>
                                <DescriptionList columnModifier={{ default: '2Col' }}>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>AddCapabilities</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            SYS_ADMIN
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Privileged</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            true
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                </DescriptionList>
                            </StackItem>
                        </Stack>
                    </DetailSection>
                </li>
                <Divider component="li" className="pf-u-mb-sm" />
                <li>
                    <DetailSection title="Port configurations">
                        @TODO: Add port configurations section
                    </DetailSection>
                </li>
            </ul>
        </div>
    );
}

export default DeploymentDetails;
