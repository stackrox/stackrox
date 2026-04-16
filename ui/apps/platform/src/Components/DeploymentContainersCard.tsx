import { Card, CardBody, CardTitle, Stack, StackItem } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
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
    const { isFeatureFlagEnabled } = useFeatureFlags();

    if (!containers || containers.length === 0) {
        return null;
    }

    const showInitContainers = isFeatureFlagEnabled('ROX_INIT_CONTAINER_SUPPORT');
    const regularContainers = showInitContainers
        ? containers.filter((c) => c.type !== 'INIT')
        : containers;
    const initContainers = showInitContainers ? containers.filter((c) => c.type === 'INIT') : [];

    return (
        <Stack hasGutter>
            {regularContainers.length > 0 && (
                <StackItem>
                    <Card>
                        <CardTitle component="h3">{title}</CardTitle>
                        <CardBody>
                            <ContainerGroup
                                containers={regularContainers}
                                getImageUrl={getImageUrl}
                            />
                        </CardBody>
                    </Card>
                </StackItem>
            )}
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
