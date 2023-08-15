import React from 'react';
import { Button, Flex, pluralize } from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';

import DeploymentScopeModal from './DeploymentScopeModal';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';
import { ClusterIcon, DeploymentIcon, NamespaceIcon } from '../common/NetworkGraphIcons';

import './NetworkPoliciesGenerationScope.css';

export type NetworkPoliciesGenerationScopeProps = {
    scopeHierarchy: NetworkScopeHierarchy;
    scopeDeploymentCount: number;
};

function NetworkPoliciesGenerationScope({
    scopeHierarchy,
    scopeDeploymentCount,
}: NetworkPoliciesGenerationScopeProps) {
    const { searchFilter } = useURLSearch();
    const isOnlyClusterScope =
        scopeHierarchy.namespaces.length === 0 && scopeHierarchy.deployments.length === 0;
    const [showDeploymentModal, setShowDeploymentModal] = React.useState(false);

    let deploymentElement = <span>All deployments</span>;
    let namespaceElement = <span>All namespaces</span>;

    if (!isOnlyClusterScope) {
        const { namespaces } = scopeHierarchy;
        const namespaceCount = namespaces.length;
        const namespaceText = namespaceCount === 1 ? namespaces[0] : `${namespaceCount} namespaces`;
        namespaceElement = <span>{namespaceText}</span>;

        const deploymentCount = scopeDeploymentCount;
        const deploymentText = pluralize(deploymentCount, 'deployment');

        deploymentElement = (
            <Button variant="link" isInline onClick={() => setShowDeploymentModal(true)}>
                {deploymentText}
            </Button>
        );
    }

    return (
        <>
            <DeploymentScopeModal
                searchFilter={searchFilter}
                scopeDeploymentCount={scopeDeploymentCount}
                isOpen={showDeploymentModal}
                onClose={() => setShowDeploymentModal(false)}
            />
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
                    <span>{scopeHierarchy.cluster.name}</span>
                </Flex>
            </div>
        </>
    );
}

export default NetworkPoliciesGenerationScope;
