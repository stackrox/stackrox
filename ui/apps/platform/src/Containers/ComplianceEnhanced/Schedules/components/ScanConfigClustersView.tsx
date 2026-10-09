import type { ReactElement } from 'react';
import { Badge, Flex, List, ListItem, Title } from '@patternfly/react-core';

import type { ComplianceIntegration } from 'services/ComplianceIntegrationService';

type ScanConfigClustersViewProps = {
    headingLevel: 'h2' | 'h3';
    clusters: ComplianceIntegration[];
};

function ScanConfigClustersView({
    headingLevel,
    clusters,
}: ScanConfigClustersViewProps): ReactElement {
    return (
        <Flex direction={{ default: 'column' }}>
            <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                <Title headingLevel={headingLevel}>Clusters</Title>
                <Badge isRead>{clusters.length}</Badge>
            </Flex>
            <List isPlain>
                {clusters.map((cluster) => (
                    <ListItem key={cluster.id}>{cluster.clusterName}</ListItem>
                ))}
            </List>
        </Flex>
    );
}

export default ScanConfigClustersView;
