import React, { useState } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    ExpandableSection,
    Text,
    TextContent,
    TextVariants,
} from '@patternfly/react-core';

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
                                    1 deploy, 1 runtime
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Processes</DescriptionListTerm>
                                <DescriptionListDescription>
                                    3 anomalous, 12 running
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
                                    0 egress, 1 ingress
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Flows observed</DescriptionListTerm>
                                <DescriptionListDescription>
                                    3 external, 2 anomalous, 4 active, 312 allowed
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </DetailSection>
                </li>
                <Divider component="li" className="pf-u-mb-sm" />
                <li>
                    <DetailSection title="Deployment configuration">
                        <DescriptionList columnModifier={{ default: '2Col' }}>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Name</DescriptionListTerm>
                                <DescriptionListDescription>
                                    visa-processor
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Cluster</DescriptionListTerm>
                                <DescriptionListDescription>Production</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Created</DescriptionListTerm>
                                <DescriptionListDescription>
                                    12/09/21 | 6:03:23 PM
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Namespace</DescriptionListTerm>
                                <DescriptionListDescription>Naples</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Replicas</DescriptionListTerm>
                                <DescriptionListDescription>2 pods</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Service account</DescriptionListTerm>
                                <DescriptionListDescription>
                                    visa-processor
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Labels</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <span>app:visa-processor</span>{' '}
                                    <span>helm.sh/release-namespace:naples</span>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Annotations</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <span>deprecated.daemonset.template.generation:15</span>{' '}
                                    <span>email:support@stackrox.com</span>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>AddCapabilities</DescriptionListTerm>
                                <DescriptionListDescription>SYS_ADMIN</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Privileged</DescriptionListTerm>
                                <DescriptionListDescription>true</DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
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
