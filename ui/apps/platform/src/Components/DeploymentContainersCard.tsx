import { Card, CardBody, CardTitle, Stack, StackItem } from '@patternfly/react-core';

import useContainerGroups from 'hooks/useContainerGroups';
import type { Container } from 'types/deployment.proto';
import DeploymentContainerConfig from './DeploymentContainerConfig';

type DeploymentContainersCardProps = {
    containers: Container[] | null | undefined;
    title: string;
    getImageUrl: (imageId: string) => string;
};

function ContainerGroup({
    containers,
    getImageUrl,
}: {
    containers: Container[];
    getImageUrl: (imageId: string) => string;
}) {
    return (
        <Stack hasGutter>
            {containers.map((container) => (
                <StackItem key={container.id}>
                    <DeploymentContainerConfig container={container} getImageUrl={getImageUrl} />
                </StackItem>
            ))}
        </Stack>
    );
}

function DeploymentContainersCard({
    containers,
    title,
    getImageUrl,
}: DeploymentContainersCardProps) {
    const { regularContainers, initContainers } = useContainerGroups(containers);

    return (
        <Stack hasGutter>
            <StackItem>
                <Card>
                    <CardTitle component="h3">{title}</CardTitle>
                    <CardBody>
                        {regularContainers.length > 0 ? (
                            <ContainerGroup
                                containers={regularContainers}
                                getImageUrl={getImageUrl}
                            />
                        ) : (
                            'None'
                        )}
                    </CardBody>
                </Card>
            </StackItem>
            {initContainers.length > 0 && (
                <StackItem>
                    <Card>
                        <CardTitle component="h3">Init container configuration</CardTitle>
                        <CardBody>
                            <ContainerGroup containers={initContainers} getImageUrl={getImageUrl} />
                        </CardBody>
                    </Card>
                </StackItem>
            )}
        </Stack>
    );
}

export default DeploymentContainersCard;
