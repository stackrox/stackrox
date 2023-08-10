import React from 'react';
import { Button, Flex, Label } from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';
import uniq from 'lodash/uniq';

import { ClusterIcon, DeploymentIcon, NamespaceIcon } from '../common/NetworkGraphIcons';

import './NetworkPoliciesGenerationScope.css';
import DeploymentScopeModal from './DeploymentScopeModal';
import { EntityScope } from '../utils/simulatorUtils';

export type NetworkPoliciesGenerationScopeProps = {
    networkPolicyGenerationScope: EntityScope;
};

function NetworkPoliciesGenerationScope({
    networkPolicyGenerationScope,
}: NetworkPoliciesGenerationScopeProps) {
    const { granularity, cluster, deployments, hasAppliedDeploymentFilters } =
        networkPolicyGenerationScope;
    const [modalEntityScope, setModalEntityScope] = React.useState<EntityScope | null>(null);

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
            <Button
                variant="link"
                isInline
                onClick={() => setModalEntityScope(networkPolicyGenerationScope)}
            >
                {deploymentText}
            </Button>
        );
    }

    return (
        <>
            {modalEntityScope && (
                <DeploymentScopeModal
                    entityScope={networkPolicyGenerationScope}
                    isOpen={modalEntityScope !== null}
                    onClose={() => setModalEntityScope(null)}
                />
            )}
            <div className="network-policies-generation-scope">
                <Flex
                    alignItems={{ default: 'alignItemsCenter' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                >
                    <DeploymentIcon aria-label="Deployment" />
                    {deploymentElement}
                    {hasAppliedDeploymentFilters && (
                        <Label isCompact icon={<FilterIcon />} color="grey">
                            Filter applied
                        </Label>
                    )}
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
