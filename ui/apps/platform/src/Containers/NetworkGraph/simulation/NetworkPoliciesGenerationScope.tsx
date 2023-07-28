import React from 'react';
import { Button, Flex } from '@patternfly/react-core';
import uniq from 'lodash/uniq';

import { ClusterIcon, DeploymentIcon, NamespaceIcon } from '../common/NetworkGraphIcons';

import './NetworkPoliciesGenerationScope.css';
import DeploymentScopeModal from './DeploymentScopeModal';

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
    const [modalDeployments, setModalDeployments] = React.useState<
        { namespace: string; name: string }[] | null
    >(null);

    let deploymentElement = <span>All deployments</span>;
    let namespaceElement = <span>All namespaces</span>;

    if (granularity !== 'CLUSTER') {
        const namespaces = uniq(deployments.map((deployment) => deployment.namespace));
        const namespaceCount = namespaces.length;
        const namespaceText = namespaceCount === 1 ? namespaces[0] : `${namespaceCount} namespaces`;
        namespaceElement = <span>{namespaceText}</span>;

        const deploymentCount = deployments.length;
        const deploymentText =
            deploymentCount === 1 ? deployments[0].name : `${deploymentCount} deployments`;

        deploymentElement = (
            <span>
                <Button
                    variant="link"
                    isInline
                    onClick={() => setModalDeployments(networkPolicyGenerationScope.deployments)}
                >
                    {deploymentText}
                </Button>
            </span>
        );
    }

    return (
        <>
            {modalDeployments && (
                <DeploymentScopeModal
                    deployments={modalDeployments}
                    isOpen={modalDeployments !== null}
                    onClose={() => setModalDeployments(null)}
                />
            )}
            <div className="network-policies-generation-scope">
                <Flex
                    alignItems={{ default: 'alignItemsCenter' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                >
                    <DeploymentIcon aria-label="Deployment" />
                    {deploymentElement}
                </Flex>

                <Flex
                    alignItems={{ default: 'alignItemsCenter' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                >
                    <NamespaceIcon aria-label="Namespace" />
                    {namespaceElement}
                </Flex>

                <Flex
                    alignItems={{ default: 'alignItemsCenter' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                >
                    <ClusterIcon aria-label="Cluster" />
                    <span>{cluster}</span>
                </Flex>
            </div>
        </>
    );
}

export default NetworkPoliciesGenerationScope;
