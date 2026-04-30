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
    return (
        <Card>
            <CardTitle component="h3">{title}</CardTitle>
            <CardBody>
                {containers && containers.length > 0 ? (
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
                ) : (
                    'None'
                )}
            </CardBody>
        </Card>
    );
}

export default DeploymentContainersCard;
