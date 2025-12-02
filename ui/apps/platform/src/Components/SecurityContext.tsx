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

import type { ContainerSecurityContext } from 'types/deployment.proto';
import { getFilteredSecurityContextMap } from 'utils/securityContextUtils';

type SecurityContextProps = {
    securityContext: ContainerSecurityContext;
};

function SecurityContext({ securityContext }: SecurityContextProps) {
    const filteredValues = getFilteredSecurityContextMap(securityContext);

    return (
        <Card>
            <CardTitle>Security context</CardTitle>
            <CardBody className="pf-v6-u-background-color-200 pf-v6-u-pt-xl pf-v6-u-mx-lg pf-v6-u-mb-lg">
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
