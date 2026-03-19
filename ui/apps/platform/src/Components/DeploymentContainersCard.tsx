import { Card, CardBody, CardTitle, Stack, StackItem } from '@patternfly/react-core';

import type { Container } from 'types/deployment.proto';
import DeploymentContainerConfig from './DeploymentContainerConfig';

type DeploymentContainersCardProps = {
    containers: Container[] | null | undefined;
    title: string;
    getImageUrl: (imageId: string) => string;
};

function DeploymentContainersCard({
    containers,
    title,
    getImageUrl,
}: DeploymentContainersCardProps) {
    if (!containers || containers.length === 0) {
        return null;
    }

    return (
        <Card>
            <CardTitle component="h3">{title}</CardTitle>
            <CardBody>
                <Stack hasGutter>
                    {containers.map((container) => (
                        <StackItem key={container.id}>
                            <DeploymentContainerConfig
                                container={container}
                                getImageUrl={getImageUrl}
                            />
                        </StackItem>
                    ))}
                </Stack>
            </CardBody>
        </Card>
    );
}

export default DeploymentContainersCard;
