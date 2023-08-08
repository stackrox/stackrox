import React from 'react';
import { Flex } from '@patternfly/react-core';
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
    const { granularity, cluster, deployments } = networkPolicyGenerationScope;

    let deploymentElement = <span>All deployments</span>;
    let namespaceElement = <span>All namespaces</span>;

    if (granularity === 'NAMESPACE' || granularity === 'DEPLOYMENT') {
        const namespaces = uniq(deployments.map((deployment) => deployment.namespace));
        const namespaceCount = namespaces.length;
        const namespaceText = namespaceCount === 1 ? namespaces[0] : `${namespaceCount} namespaces`;
        namespaceElement = <span>{namespaceText}</span>;
    }

    if (granularity === 'DEPLOYMENT') {
        const deploymentCount = deployments.length;
        const deploymentText =
            deploymentCount === 1 ? deployments[0].name : `${deploymentCount} deployments`;

        deploymentElement = <span>{deploymentText}</span>;
    }

    return (
        <div className="network-policies-generation-scope">
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <DeploymentIcon title="Deployment" />
                {deploymentElement}
            </Flex>

            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <NamespaceIcon title="Namespace" />
                {namespaceElement}
            </Flex>

            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <ClusterIcon title="Cluster" />
                <span>{cluster}</span>
            </Flex>
        </div>
    );
}

export default NetworkPoliciesGenerationScope;
