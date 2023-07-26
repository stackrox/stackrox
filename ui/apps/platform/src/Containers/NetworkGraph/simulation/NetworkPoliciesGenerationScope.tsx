import React from 'react';
import { Flex } from '@patternfly/react-core';
import pluralize from 'pluralize';
import uniq from 'lodash/uniq';

import { ClusterIcon, DeploymentIcon, NamespaceIcon } from '../common/NetworkGraphIcons';

import './NetworkPoliciesGenerationScope.css';

export type EntityScope = {
    // `granularity` refers to the most specific entity type that has been selected by the user.
    granularity: 'CLUSTER' | 'NAMESPACE' | 'DEPLOYMENT';
    cluster: string;
    namespaces: string[];
    deployments: {
        namespace: string;
        name: string;
    }[];
};

export type NetworkPoliciesGenerationScopeProps = {
    networkPolicyGenerationScope: EntityScope;
};

function NetworkPoliciesGenerationScope({
    networkPolicyGenerationScope,
}: NetworkPoliciesGenerationScopeProps) {
    const { granularity, cluster } = networkPolicyGenerationScope;

    let deploymentElement = <span>All deployments</span>;
    let namespaceElement = <span>All namespaces</span>;

    if (granularity === 'NAMESPACE') {
        const namespaceCount = networkPolicyGenerationScope.namespaces.length;
        namespaceElement = (
            <span>
                {namespaceCount} {pluralize('namespace', namespaceCount)}
            </span>
        );
    } else if (granularity === 'DEPLOYMENT') {
        // Only count namespaces that have a deployment in scope, even if the namespace is selected
        // otherwise
        const namespaceCount = uniq(
            networkPolicyGenerationScope.deployments.map((deployment) => deployment.namespace)
        ).length;
        namespaceElement = (
            <span>
                {namespaceCount} {pluralize('namespace', namespaceCount)}
            </span>
        );
        const deploymentCount = networkPolicyGenerationScope.deployments.length;
        deploymentElement = (
            <span>
                {deploymentCount} {pluralize('deployment', deploymentCount)}
            </span>
        );
    }

    return (
        <div className="network-policies-generation-scope">
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <DeploymentIcon />
                {deploymentElement}
            </Flex>

            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <NamespaceIcon />
                {namespaceElement}
            </Flex>

            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <ClusterIcon />
                <span>cluster &quot;{cluster}&quot;</span>
            </Flex>
        </div>
    );
}

export default NetworkPoliciesGenerationScope;
