import React from 'react';

import NetworkPolicies from '../common/NetworkPolicies';

function NamespaceNetworkPolicies() {
    // @TODO: We will eventually do an API call to fetch the network policies based on the
    // network policy ids for the selected node
    const networkPolicies = [
        {
            name: 'payments-network-rules',
            yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ''
  namespace: managed-service-registry`,
        },
        {
            name: 'test-network-rules',
            yaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ''
  namespace: test-service-registry`,
        },
    ];

    return (
        <div className="pf-u-h-100 pf-u-p-md">
            <NetworkPolicies networkPolicies={networkPolicies} />
        </div>
    );
}

export default NamespaceNetworkPolicies;
