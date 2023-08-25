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

import { ContainerVolume } from 'types/deployment.proto';

type ContainerVolumeInfoProps = {
    volumes: ContainerVolume[];
};

function ContainerVolumeInfo({ volumes }: ContainerVolumeInfoProps) {
    const initialToggleValues = Array.from({ length: volumes.length }, () => true);
    const [volumeToggles, setVolumeToggles] = useState(initialToggleValues);

    function setToggleAtIndex(i) {
        const newToggles = [...volumeToggles];
        newToggles[i] = !newToggles[i];

        setVolumeToggles(newToggles);
    }

    return (
        <Card>
            <CardTitle>Volumes</CardTitle>
            <CardBody>
                <Stack hasGutter>
                    {volumes.length > 0 ? (
                        volumes.map((volume, index) => (
                            <StackItem>
                                <ExpandableSection
                                    toggleText={volume.name}
                                    onToggle={() => setToggleAtIndex(index)}
                                    isExpanded={volumeToggles[index]}
                                    className="pf-expandable-not-large"
                                >
                                    <DescriptionList
                                        columnModifier={{ default: '2Col' }}
                                        isCompact
                                        className="pf-u-background-color-200 pf-u-p-md"
                                    >
                                        <DescriptionListGroup>
                                            <DescriptionListTerm>Source</DescriptionListTerm>
                                            <DescriptionListDescription>
                                                {volume.source || '-'}
                                            </DescriptionListDescription>
                                        </DescriptionListGroup>
                                        <DescriptionListGroup>
                                            <DescriptionListTerm>Destination</DescriptionListTerm>
                                            <DescriptionListDescription>
                                                {volume.destination || '-'}
                                            </DescriptionListDescription>
                                        </DescriptionListGroup>
                                        <DescriptionListGroup>
                                            <DescriptionListTerm>Read only</DescriptionListTerm>
                                            <DescriptionListDescription>
                                                {volume.readOnly || 'false'}
                                            </DescriptionListDescription>
                                        </DescriptionListGroup>
                                        <DescriptionListGroup>
                                            <DescriptionListTerm>Type</DescriptionListTerm>
                                            <DescriptionListDescription>
                                                {volume.type}
                                            </DescriptionListDescription>
                                        </DescriptionListGroup>
                                        <DescriptionListGroup>
                                            <DescriptionListTerm>
                                                Mount propagation
                                            </DescriptionListTerm>
                                            <DescriptionListDescription>
                                                {volume.mountPropagation}
                                            </DescriptionListDescription>
                                        </DescriptionListGroup>
                                    </DescriptionList>
                                </ExpandableSection>
                            </StackItem>
                        ))
                    ) : (
                        <EmptyState>No volumes</EmptyState>
                    )}
                </Stack>
            </CardBody>
        </Card>
    );
}

export default ContainerVolumeInfo;
