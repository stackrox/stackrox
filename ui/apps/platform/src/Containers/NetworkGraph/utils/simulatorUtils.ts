import { NetworkPolicyModification } from 'types/networkPolicy.proto';
import { EntityScope } from '../simulation/NetworkPoliciesGenerationScope';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';
import { CustomNodeModel } from '../types/topology.type';

export function getDisplayYAMLFromNetworkPolicyModification(
    modification: NetworkPolicyModification | null
): string {
    const { applyYaml, toDelete } = modification || {};
    const shouldDelete = toDelete && toDelete.length > 0;
    const showApplyYaml = applyYaml && applyYaml.length >= 2;

    const toDeleteSection = shouldDelete
        ? toDelete
              .map((entry) => `# kubectl -n ${entry.namespace} delete networkpolicy ${entry.name}`)
              .join('\n')
        : '';

    // Format complete YAML for display.
    let displayYaml: string;
    if (shouldDelete && showApplyYaml) {
        displayYaml = [toDeleteSection, applyYaml].join('\n---\n');
    } else if (shouldDelete && !showApplyYaml) {
        displayYaml = toDeleteSection;
    } else if (!shouldDelete && showApplyYaml) {
        displayYaml = applyYaml;
    } else {
        displayYaml = 'No policies need to be created or deleted.';
    }

    return displayYaml;
}

function getGranularityFromScopeHierarchy(
    scopeHierarchy: NetworkScopeHierarchy
): EntityScope['granularity'] {
    if (scopeHierarchy.namespaces.length === 0 && scopeHierarchy.deployments.length === 0) {
        // Only a cluster is selected
        return 'CLUSTER';
    }
    if (scopeHierarchy.namespaces.length > 0 && scopeHierarchy.deployments.length === 0) {
        // A cluster and at least one namespace is selected
        return 'NAMESPACE';
    }
    // A cluster, at least one namespace, and at least one deployment is selected
    return 'DEPLOYMENT';
}

/**
 * Given the array of nodeModels returned by the network graph API and the user's
 * current scope hierarchy, return the scope of entities that should be included
 * in the network policy simulation.
 */
export function getInScopeEntities(
    nodeModels: CustomNodeModel[],
    scopeHierarchy: NetworkScopeHierarchy
): EntityScope {
    const granularity = getGranularityFromScopeHierarchy(scopeHierarchy);

    // Include all deployments from the model that match a selected namespace and
    // selected deployment name from the scope hierarchy. This prevents the inclusion
    // of namespaces and deployments that are visible due to active connections between nodes
    // and not due to the user's selection.
    const namespaceNameSet = new Set(scopeHierarchy.namespaces);
    const deploymentNameSet = new Set(scopeHierarchy.deployments);

    const deploymentScope: EntityScope = {
        granularity,
        cluster: scopeHierarchy.cluster.name,
        namespaces: scopeHierarchy.namespaces,
        deployments: [],
    };

    nodeModels.forEach((node) => {
        if (node.data.type !== 'DEPLOYMENT') {
            return;
        }

        const inSelectedNamespaces = namespaceNameSet.has(node.data.deployment.namespace);
        const inSelectedDeployments = deploymentNameSet.has(node.data.deployment.name);
        if (
            (granularity === 'NAMESPACE' && inSelectedNamespaces) ||
            (granularity === 'DEPLOYMENT' && inSelectedNamespaces && inSelectedDeployments)
        ) {
            deploymentScope.deployments.push(node.data.deployment);
        }
    });

    return deploymentScope;
}
