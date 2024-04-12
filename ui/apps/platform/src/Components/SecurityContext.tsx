import React from 'react';
import {
    Card,
    CardBody,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    EmptyState,
} from '@patternfly/react-core';

import { ContainerSecurityContext } from 'types/deployment.proto';
import { getFilteredSecurityContextMap } from 'utils/securityContextUtils';

type SecurityContextProps = {
    securityContext: ContainerSecurityContext;
};

function SecurityContext({ securityContext }: SecurityContextProps) {
    const filteredValues = getFilteredSecurityContextMap(securityContext);

    return (
        <Card>
            <CardTitle>Security context</CardTitle>
            <CardBody className="pf-v5-u-background-color-200 pf-v5-u-pt-xl pf-v5-u-mx-lg pf-v5-u-mb-lg">
                {filteredValues.size > 0 ? (
                    <DescriptionList columnModifier={{ default: '2Col' }} isCompact>
                        {Array.from(filteredValues.entries()).map(([key, value]) => {
                            return (
                                <DescriptionListGroup key={key}>
                                    <DescriptionListTerm>{key}</DescriptionListTerm>
                                    <DescriptionListDescription>{value}</DescriptionListDescription>
                                </DescriptionListGroup>
                            );
                        })}
                    </DescriptionList>
                ) : (
                    <EmptyState>No container security context</EmptyState>
                )}
            </CardBody>
        </Card>
    );
}

export default SecurityContext;
