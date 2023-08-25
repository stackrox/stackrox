import React, { useState } from 'react';
import {
    Card,
    CardBody,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    EmptyState,
    ExpandableSection,
    Stack,
    StackItem,
} from '@patternfly/react-core';

import { EmbeddedSecret } from 'types/deployment.proto';

type ContainerSecretInfoProps = {
    secrets: EmbeddedSecret[];
};

function ContainerSecretInfo({ secrets }: ContainerSecretInfoProps) {
    const initialToggleValues = Array.from({ length: secrets.length }, () => true);
    const [secretToggles, setSecretToggles] = useState(initialToggleValues);

    function setToggleAtIndex(i) {
        const newToggles = [...secretToggles];
        newToggles[i] = !newToggles[i];

        setSecretToggles(newToggles);
    }

    return (
        <Card>
            <CardTitle>Secrets</CardTitle>
            <CardBody>
                <Stack hasGutter>
                    {secrets.length > 0 ? (
                        secrets.map((secret, index) => (
                            <StackItem>
                                <ExpandableSection
                                    toggleText={secret.name}
                                    onToggle={() => setToggleAtIndex(index)}
                                    isExpanded={secretToggles[index]}
                                    className="pf-expandable-not-large"
                                >
                                    <DescriptionList
                                        isCompact
                                        className="pf-u-background-color-200 pf-u-p-md"
                                    >
                                        <DescriptionListGroup>
                                            <DescriptionListTerm>Source</DescriptionListTerm>
                                            <DescriptionListDescription>
                                                {secret.path}
                                            </DescriptionListDescription>
                                        </DescriptionListGroup>
                                    </DescriptionList>
                                </ExpandableSection>
                            </StackItem>
                        ))
                    ) : (
                        <EmptyState>No secrets</EmptyState>
                    )}
                </Stack>
            </CardBody>
        </Card>
    );
}

export default ContainerSecretInfo;
