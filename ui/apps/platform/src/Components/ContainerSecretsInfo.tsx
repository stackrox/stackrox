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

import type { EmbeddedSecret } from 'types/deployment.proto';

type ContainerSecretsInfoProps = {
    secrets: EmbeddedSecret[];
};

function ContainerSecretsInfo({ secrets }: ContainerSecretsInfoProps) {
    return (
        <Card>
            <CardTitle>Secrets</CardTitle>
            <CardBody>
                {secrets.length > 0 ? (
                    <Stack hasGutter>
                        {secrets.map((secret, index) => (
                            <StackItem key={secret.name}>
                                <Stack hasGutter>
                                    <StackItem>
                                        <Label color="blue" isCompact>
                                            {secret.name}
                                        </Label>
                                    </StackItem>
                                    <StackItem>
                                        <DescriptionList isCompact>
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>Path</DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {secret.path}
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                        </DescriptionList>
                                    </StackItem>
                                    {index < secrets.length - 1 && (
                                        <StackItem>
                                            <Divider />
                                        </StackItem>
                                    )}
                                </Stack>
                            </StackItem>
                        ))}
                    </Stack>
                ) : (
                    <EmptyState>No secrets</EmptyState>
                )}
            </CardBody>
        </Card>
    );
}

export default ContainerSecretsInfo;
