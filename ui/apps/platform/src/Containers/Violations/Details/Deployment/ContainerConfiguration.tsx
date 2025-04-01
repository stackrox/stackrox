import React, { ReactElement } from 'react';
import { Card, CardBody, CardTitle, Title } from '@patternfly/react-core';

import { Deployment } from 'types/deployment.proto';
import useFeatureFlags from 'hooks/useFeatureFlags';
import {
    vulnerabilitiesPlatformPath,
    vulnerabilitiesUserWorkloadsPath,
    vulnerabilitiesWorkloadCvesPath,
} from 'routePaths';
import ContainerConfigurationDescriptionList from './ContainerConfigurationDescriptionList';

export type ContainerConfigurationProps = {
    deployment: Deployment | null;
};

function ContainerConfiguration({ deployment }: ContainerConfigurationProps): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const hasPlatformWorkloadCveLink = deployment && deployment.platformComponent;

    // eslint-disable-next-line no-nested-ternary
    const vulnMgmtBasePath = !isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')
        ? vulnerabilitiesWorkloadCvesPath
        : hasPlatformWorkloadCveLink
          ? vulnerabilitiesPlatformPath
          : vulnerabilitiesUserWorkloadsPath;

    let content: JSX.Element[] | string = 'None';

    if (deployment === null) {
        content =
            'Container configurations are unavailable because the alert’s deployment no longer exists.';
    } else if (deployment.containers.length !== 0) {
        content = deployment.containers.map((container, i) => (
            <React.Fragment key={container.id}>
                <Title headingLevel="h4" className="pf-v5-u-mb-md">{`containers[${i}]`}</Title>
                <ContainerConfigurationDescriptionList
                    key={container.id}
                    container={container}
                    vulnMgmtBasePath={vulnMgmtBasePath}
                />
            </React.Fragment>
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
