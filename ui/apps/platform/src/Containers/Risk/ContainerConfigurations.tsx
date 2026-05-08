import { vulnerabilitiesPlatformPath, vulnerabilitiesUserWorkloadsPath } from 'routePaths';

import type { Deployment } from 'types/deployment.proto';
import DeploymentContainersCard from 'Components/DeploymentContainersCard';

type ContainerConfigurationsProps = {
    deployment: Deployment;
};

function ContainerConfigurations({ deployment }: ContainerConfigurationsProps) {
    const vulnMgmtBasePath = deployment?.platformComponent
        ? vulnerabilitiesPlatformPath
        : vulnerabilitiesUserWorkloadsPath;

    const getImageUrl = (imageId: string) => `${vulnMgmtBasePath}/images/${imageId}`;

    return (
        <DeploymentContainersCard
            containers={deployment.containers}
            title="Container configuration"
            getImageUrl={getImageUrl}
        />
    );
}

export default ContainerConfigurations;
