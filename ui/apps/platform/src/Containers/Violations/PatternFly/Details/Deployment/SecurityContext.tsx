/* eslint-disable @typescript-eslint/naming-convention */
import React, { ReactElement } from 'react';
import { Card, CardBody, DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';

function SecurityContext({ deployment }): ReactElement {
    const securityContextContainers = deployment?.containers?.filter(
        (container) => !!container.securityContext
    );
    return (
        <Card isFlat data-testid="security-context">
            <CardBody>
                {securityContextContainers?.length > 0
                    ? securityContextContainers.map((container) => {
                          const {
                              privileged,
                              addCapabilities,
                              dropCapabilities,
                          } = container.securityContext;
                          return (
                              <DescriptionList isHorizontal>
                                  {privileged && (
                                      <DescriptionListItem term="Privileged" desc="true" />
                                  )}
                                  {addCapabilities.length > 0 && (
                                      <DescriptionListItem
                                          term="AddC apabilities"
                                          desc={addCapabilities}
                                      />
                                  )}
                                  {dropCapabilities.length > 0 && (
                                      <DescriptionListItem
                                          term="Drop Capabilities"
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
