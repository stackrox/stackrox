import { Fragment } from 'react';
import type { ReactElement } from 'react';
import { Card, CardBody, CardTitle, Title } from '@patternfly/react-core';

import type { Deployment } from 'types/deployment.proto';
import { vulnerabilitiesPlatformPath, vulnerabilitiesUserWorkloadsPath } from 'routePaths';
import ContainerConfigurationDescriptionList from './ContainerConfigurationDescriptionList';

export type ContainerConfigurationProps = {
    deployment: Deployment | null;
};

function ContainerConfiguration({ deployment }: ContainerConfigurationProps): ReactElement {
    const vulnMgmtBasePath = deployment?.platformComponent
        ? vulnerabilitiesPlatformPath
        : vulnerabilitiesUserWorkloadsPath;

    let content: JSX.Element[] | string = 'None';

    if (deployment === null) {
        content =
            'Container configurations are unavailable because the alert’s deployment no longer exists.';
    } else if (deployment.containers.length !== 0) {
        content = deployment.containers.map((container, i) => (
            <Fragment key={container.id}>
                <Title headingLevel="h4" className="pf-v5-u-mb-md">{`containers[${i}]`}</Title>
                <ContainerConfigurationDescriptionList
                    key={container.id}
                    container={container}
                    vulnMgmtBasePath={vulnMgmtBasePath}
                />
            </Fragment>
        ));
    }

    return (
        <Card isFlat>
            <CardTitle component="h3">Container configuration</CardTitle>
            <CardBody>{content}</CardBody>
        </Card>
    );
}

export default ContainerConfiguration;
