import {
    Card,
    CardBody,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    EmptyState,
    Label,
    Stack,
    StackItem,
} from '@patternfly/react-core';

import type { ContainerVolume } from 'types/deployment.proto';

type ContainerVolumesInfoProps = {
    volumes: ContainerVolume[];
};

function ContainerVolumesInfo({ volumes }: ContainerVolumesInfoProps) {
    return (
        <Card>
            <CardTitle>Volumes</CardTitle>
            <CardBody>
                {volumes.length > 0 ? (
                    <Stack hasGutter>
                        {volumes.map((volume, index) => (
                            <StackItem key={volume.name}>
                                <Stack hasGutter>
                                    <StackItem>
                                        <Label color="blue" isCompact>
                                            {volume.name}
                                        </Label>
                                    </StackItem>
                                    <StackItem>
                                        <DescriptionList
                                            columnModifier={{ default: '2Col' }}
                                            isCompact
                                        >
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>Source</DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {volume.source || '-'}
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>
                                                    Destination
                                                </DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {volume.destination || '-'}
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>Read only</DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {volume.readOnly ? 'true' : 'false'}
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
                                    </StackItem>
                                    {index < volumes.length - 1 && (
                                        <StackItem>
                                            <Divider />
                                        </StackItem>
                                    )}
                                </Stack>
                            </StackItem>
                        ))}
                    </Stack>
                ) : (
                    <EmptyState>No volumes</EmptyState>
                )}
            </CardBody>
        </Card>
    );
}

export default ContainerVolumesInfo;
