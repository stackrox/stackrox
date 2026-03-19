import { Card, CardBody, CardTitle, DescriptionList } from '@patternfly/react-core';

import type { Container } from 'types/deployment.proto';
import DescriptionListItem from 'Components/DescriptionListItem';

type SecurityContextCardProps = {
    containers?: Container[] | null;
    emptyMessage?: string;
    headingComponent?: 'h2' | 'h3' | 'h4';
};

function SecurityContextCard({
    containers = [],
    emptyMessage = 'None',
    headingComponent = 'h3',
}: SecurityContextCardProps) {
    const securityContextContainers = (containers ?? []).filter(
        (container) =>
            !!(
                container?.securityContext?.privileged ||
                container?.securityContext?.addCapabilities.length > 0 ||
                container?.securityContext?.dropCapabilities.length > 0
            )
    );

    if (securityContextContainers.length === 0) {
        return (
            <Card>
                <CardTitle component={headingComponent}>Security context</CardTitle>
                <CardBody>{emptyMessage}</CardBody>
            </Card>
        );
    }

    const content = securityContextContainers.map((container) => {
        const { privileged, addCapabilities, dropCapabilities } = container.securityContext;
        return (
            <DescriptionList isHorizontal key={container.id}>
                {privileged && <DescriptionListItem term="Privileged" desc="true" />}
                {addCapabilities.length > 0 && (
                    <DescriptionListItem term="Add capabilities" desc={addCapabilities} />
                )}
                {dropCapabilities.length > 0 && (
                    <DescriptionListItem term="Drop capabilities" desc={dropCapabilities} />
                )}
            </DescriptionList>
        );
    });

    return (
        <Card>
            <CardTitle component={headingComponent}>Security context</CardTitle>
            <CardBody>{content}</CardBody>
        </Card>
    );
}

export default SecurityContextCard;
