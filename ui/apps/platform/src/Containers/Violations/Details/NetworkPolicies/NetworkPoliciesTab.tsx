import React, { ReactElement, useEffect, useState } from 'react';
import { Button, Card, CardBody, CardTitle, List, ListItem } from '@patternfly/react-core';

import { fetchNetworkPoliciesInNamespace } from 'services/NetworkService';
import { NetworkPolicy } from 'types/networkPolicy.proto';

import NetworkPolicyModal from './NetworkPolicyModal';

const compareNetworkPolicies = (a: NetworkPolicy, b: NetworkPolicy): number => {
    return a.name.localeCompare(b.name);
};

export type NetworkPoliciesTabProps = {
    clusterId: string;
    namespaceName: string;
};

function NetworkPoliciesTab({ clusterId, namespaceName }: NetworkPoliciesTabProps): ReactElement {
    const [selectedNetworkPolicy, setSelectedNetworkPolicy] = useState<NetworkPolicy | null>(null);
    const [namespacePolicies, setNamespacePolicies] = useState<NetworkPolicy[]>([]);

    useEffect(() => {
        fetchNetworkPoliciesInNamespace(clusterId, namespaceName).then(
            // TODO Infer type from response once NetworkService.js is typed
            (policies: NetworkPolicy[]) => setNamespacePolicies(policies ?? []),
            () => setNamespacePolicies([])
        );
    }, [clusterId, namespaceName, setNamespacePolicies]);

    return (
        <Card isFlat>
            <CardTitle component="h3">Network policies</CardTitle>
            <CardBody>
                <div className="pf-u-mb-md">{`Namespace: ${namespaceName}`}</div>
                {namespacePolicies.length > 0 ? (
                    <>
                        {selectedNetworkPolicy && (
                            <NetworkPolicyModal
                                networkPolicy={selectedNetworkPolicy}
                                isOpen={selectedNetworkPolicy !== null}
                                onClose={() => setSelectedNetworkPolicy(null)}
                            />
                        )}
                        <List component="ol">
                            {namespacePolicies
                                .sort(compareNetworkPolicies)
                                .map((netpol: NetworkPolicy) => (
                                    <ListItem>
                                        <Button
                                            key={netpol.id}
                                            variant="link"
                                            onClick={() => setSelectedNetworkPolicy(netpol)}
                                        >
                                            {netpol.name}
                                        </Button>
                                    </ListItem>
                                ))}
                        </List>
                    </>
                ) : (
                    <>Namespace has no network policies</>
                )}
            </CardBody>
        </Card>
    );
}

export default NetworkPoliciesTab;
