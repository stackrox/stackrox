import React, { ReactElement } from 'react';
import { Alert, Card, CardBody, CardTitle, Flex, FlexItem, Title } from '@patternfly/react-core';

import useFetchDeployment from 'hooks/useFetchDeployment';
import usePermissions from 'hooks/usePermissions';
import { DeploymentAlert } from 'types/alert.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ContainerConfiguration from './ContainerConfiguration';
import DeploymentOverview from './DeploymentOverview';
import NetworkPoliciesCard from './NetworkPoliciesCard';
import PortDescriptionList from './PortDescriptionList';
import SecurityContext from './SecurityContext';

export type DeploymentDetailsProps = {
    alertDeployment: Pick<
        NonNullable<DeploymentAlert['deployment']>,
        'id' | 'clusterId' | 'namespace'
    >;
};

function DeploymentDetails({ alertDeployment }: DeploymentDetailsProps): ReactElement {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForNetworkPolicy = hasReadAccess('NetworkPolicy');

    // attempt to fetch related deployment to selected alert
    const { deployment: relatedDeployment, error: relatedDeploymentFetchError } =
        useFetchDeployment(alertDeployment.id);

    const relatedDeploymentPorts = relatedDeployment?.ports || [];

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
                    title="There was an error fetching the deployment details. This deployment may no longer exist."
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
                                {relatedDeployment && (
                                    <DeploymentOverview deployment={relatedDeployment} />
                                )}
                            </CardBody>
                        </Card>
                    </FlexItem>
                    <FlexItem>
                        <Card isFlat>
                            <CardTitle component="h3">Port configuration</CardTitle>
                            <CardBody>
                                {relatedDeploymentPorts.length === 0
                                    ? 'None'
                                    : relatedDeploymentPorts.map((port, i) => {
                                          /* eslint-disable react/no-array-index-key */
                                          return (
                                              <React.Fragment key={i}>
                                                  <Title
                                                      headingLevel="h4"
                                                      className="pf-u-mb-md"
                                                  >{`ports[${i}]`}</Title>
                                                  <PortDescriptionList port={port} />
                                              </React.Fragment>
                                          );
                                          /* eslint-enable react/no-array-index-key */
                                      })}
                            </CardBody>
                        </Card>
                    </FlexItem>
                    <FlexItem>
                        <SecurityContext deployment={relatedDeployment} />
                    </FlexItem>
                    {hasReadAccessForNetworkPolicy && (
                        <FlexItem>
                            <NetworkPoliciesCard
                                clusterId={alertDeployment.clusterId}
                                namespaceName={alertDeployment.namespace}
                            />
                        </FlexItem>
                    )}
                </Flex>
                <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                    <FlexItem>
                        <Card isFlat>
                            <CardTitle component="h3">Container configuration</CardTitle>
                            <CardBody>
                                {Array.isArray(relatedDeployment?.containers) &&
                                relatedDeployment?.containers.length !== 0
                                    ? relatedDeployment?.containers.map((container, i) => (
                                          <React.Fragment key={container.id}>
                                              <Title
                                                  headingLevel="h4"
                                                  className="pf-u-mb-md"
                                              >{`containers[${i}]`}</Title>
                                              <ContainerConfiguration
                                                  key={container.id}
                                                  container={container}
                                              />
                                          </React.Fragment>
                                      ))
                                    : 'None'}
                            </CardBody>
                        </Card>
                    </FlexItem>
                </Flex>
            </Flex>
        </Flex>
    );
}

export default DeploymentDetails;
