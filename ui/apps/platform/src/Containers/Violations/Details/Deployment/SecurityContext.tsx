import React, { ReactElement } from 'react';
import { Card, CardBody, DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { Deployment } from 'types/deployment.proto';

export type SecurityContextProps = {
    deployment: Deployment | null;
};

function SecurityContext({ deployment }: SecurityContextProps): ReactElement {
    const securityContextContainers =
        deployment?.containers?.filter(
            (container) =>
                !!(
                    container?.securityContext?.privileged ||
                    container?.securityContext?.addCapabilities.length > 0 ||
                    container?.securityContext?.dropCapabilities.length > 0
                )
        ) ?? [];
    return (
        <Card isFlat aria-label="Security context">
            <CardBody>
                {securityContextContainers?.length > 0
                    ? securityContextContainers.map((container, idx) => {
                          const { privileged, addCapabilities, dropCapabilities } =
                              container.securityContext;
                          return (
                              // eslint-disable-next-line react/no-array-index-key
                              <DescriptionList isHorizontal key={idx}>
                                  {privileged && (
                                      <DescriptionListItem term="Privileged" desc="true" />
                                  )}
                                  {addCapabilities.length > 0 && (
                                      <DescriptionListItem
                                          term="Add capabilities"
                                          desc={addCapabilities}
                                      />
                                  )}
                                  {dropCapabilities.length > 0 && (
                                      <DescriptionListItem
                                          term="Drop capabilities"
                                          desc={dropCapabilities}
                                      />
                                  )}
                              </DescriptionList>
                          );
                      })
                    : 'None'}
            </CardBody>
        </Card>
    );
}

export default SecurityContext;
