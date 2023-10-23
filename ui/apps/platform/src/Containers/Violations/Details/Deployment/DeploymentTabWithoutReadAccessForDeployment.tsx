import React, { ReactElement } from 'react';
import { Card, CardBody, CardTitle, Flex, FlexItem } from '@patternfly/react-core';

import { AlertDeployment } from 'types/alert.proto';

import DeploymentOverview from './DeploymentOverview';

export type DeploymentTabWithoutReadAccessForDeploymentProps = {
    alertDeployment: AlertDeployment;
};

function DeploymentTabWithoutReadAccessForDeployment({
    alertDeployment,
}: DeploymentTabWithoutReadAccessForDeploymentProps): ReactElement {
    return (
        <Flex
            direction={{ default: 'column' }}
            flex={{ default: 'flex_1' }}
            aria-label="Deployment details"
        >
            <Flex flex={{ default: 'flex_1' }}>
                <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                    <FlexItem>
                        <Card isFlat>
                            <CardTitle component="h3">Deployment overview</CardTitle>
                            <CardBody>
                                <DeploymentOverview alertDeployment={alertDeployment} />
                            </CardBody>
                        </Card>
                    </FlexItem>
                </Flex>
            </Flex>
        </Flex>
    );
}

export default DeploymentTabWithoutReadAccessForDeployment;
