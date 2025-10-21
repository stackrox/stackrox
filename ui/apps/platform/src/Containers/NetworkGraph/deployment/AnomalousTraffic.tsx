import {
    Alert,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    Spinner,
} from '@patternfly/react-core';

import useFetchNetworkFlows from '../api/useFetchNetworkFlows';
import type { EdgeState } from '../components/EdgeStateSelect';
import type { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';

import AnomalousFlows from './AnomalousFlows';

export type AnomalousTrafficProps = {
    deploymentId: string;
    edgeState: EdgeState;
    edges: CustomEdgeModel[];
    nodes: CustomNodeModel[];
};

function AnomalousTraffic({ deploymentId, edgeState, edges, nodes }: AnomalousTrafficProps) {
    const {
        isLoading,
        error,
        data: { networkFlows },
    } = useFetchNetworkFlows({ deploymentId, edgeState, edges, nodes });

    return (
        <DescriptionListGroup>
            <DescriptionListTerm>Anomalous traffic</DescriptionListTerm>
            <DescriptionListDescription>
                <Flex
                    direction={{ default: 'row' }}
                    justifyContent={{ default: 'justifyContentCenter' }}
                >
                    {isLoading ? (
                        <Spinner />
                    ) : error ? (
                        <Alert
                            variant="warning"
                            title="Unable to fetch network flows"
                            isInline
                            component="p"
                            className="pf-v5-u-w-100"
                        >
                            {error}
                        </Alert>
                    ) : (
                        <AnomalousFlows networkFlows={networkFlows} />
                    )}
                </Flex>
            </DescriptionListDescription>
        </DescriptionListGroup>
    );
}

export default AnomalousTraffic;
