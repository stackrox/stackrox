import type { ReactElement } from 'react';
import { Alert, Card, CardBody, CardTitle, Flex, FlexItem } from '@patternfly/react-core';

import useFetchDeployment from 'hooks/useFetchDeployment';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import type { AlertDeployment } from 'types/alert.proto';

import DeploymentOverview from './DeploymentOverview';
import SecurityContext from './SecurityContext';
import PortConfiguration from './PortConfiguration';
import ContainerConfiguration from './ContainerConfiguration';

export type DeploymentTabWithReadAccessForDeploymentProps = {
    alertDeployment: AlertDeployment;
};

function DeploymentTabWithReadAccessForDeployment({
    alertDeployment,
}: DeploymentTabWithReadAccessForDeploymentProps): ReactElement {
    // attempt to fetch related deployment to selected alert
    const { deployment: relatedDeployment, error: relatedDeploymentFetchError } =
        useFetchDeployment(alertDeployment.id);

    return (
        <Flex
            direction={{ default: 'column' }}
            flex={{ default: 'flex_1' }}
            aria-label="Deployment details"
        >
            {!relatedDeployment && relatedDeploymentFetchError && (
                <Alert
                    variant="warning"
                    isInline
                    title="Unable to fetch deployment details. The deployment may no longer exist. The information below reflects the last known state at the time the alert was triggered and may be outdated."
                    component="p"
                >
                    {getAxiosErrorMessage(relatedDeploymentFetchError)}
                </Alert>
            )}
            <Flex flex={{ default: 'flex_1' }}>
                <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                    <FlexItem>
                        <Card isFlat>
                            <CardTitle component="h3">Deployment overview</CardTitle>
                            <CardBody>
                                <DeploymentOverview
                                    alertDeployment={alertDeployment}
                                    deployment={relatedDeployment}
                                />
                            </CardBody>
                        </Card>
                    </FlexItem>
                    <FlexItem>
                        <PortConfiguration deployment={relatedDeployment} />
                    </FlexItem>
                    <FlexItem>
                        <SecurityContext deployment={relatedDeployment} />
                    </FlexItem>
                </Flex>
                <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                    <FlexItem>
                        <ContainerConfiguration deployment={relatedDeployment} />
                    </FlexItem>
                </Flex>
            </Flex>
        </Flex>
    );
}

export default DeploymentTabWithReadAccessForDeployment;
