import type { ReactElement } from 'react';
import { Card, CardBody, CardTitle } from '@patternfly/react-core';

import type { Deployment } from 'types/deployment.proto';
import { vulnerabilitiesPlatformPath, vulnerabilitiesUserWorkloadsPath } from 'routePaths';
import DeploymentContainersCard from 'Components/DeploymentContainersCard';

export type ContainerConfigurationProps = {
    deployment: Deployment | null;
};

function ContainerConfiguration({ deployment }: ContainerConfigurationProps): ReactElement {
    const vulnMgmtBasePath = deployment?.platformComponent
        ? vulnerabilitiesPlatformPath
        : vulnerabilitiesUserWorkloadsPath;

    const getImageUrl = (imageId: string) => `${vulnMgmtBasePath}/images/${imageId}`;

    if (deployment === null) {
        return (
            <Card>
                <CardTitle component="h3">Container configuration</CardTitle>
                <CardBody>
                    Container configurations are unavailable because the alert&apos;s deployment no
                    longer exists.
                </CardBody>
            </Card>
        );
    }

    return (
        <DeploymentContainersCard
            containers={deployment.containers}
            title="Container configuration"
            getImageUrl={getImageUrl}
        />
    );
}

export default ContainerConfiguration;
