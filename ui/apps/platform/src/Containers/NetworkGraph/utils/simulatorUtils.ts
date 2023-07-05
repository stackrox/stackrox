import pluralize from 'pluralize';
import { NetworkPolicyModification } from 'types/networkPolicy.proto';
import { ensureExhaustive } from 'utils/type.utils';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

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

export function getSimulationPanelHeaderText(
    scopeHierarchy: Omit<NetworkScopeHierarchy, 'remainingQuery'>
): string {
    const scopeTextFormat = getScopeTextFormat(scopeHierarchy);
    const cluster = scopeHierarchy.cluster.name;

    switch (scopeTextFormat.type) {
        case 'CLUSTER_ONLY':
            return `Simulate network policies for all deployments in cluster "${cluster}"`;
        case 'FEW_NAMESPACES': {
            const namespaceText = pluralize('namespace', scopeTextFormat.namespaces.length);
            const namespaces = scopeTextFormat.namespaces.join('", "');
            return `Simulate network policies for all deployments in ${namespaceText} "${namespaces}" in cluster "${cluster}"`;
        }
        case 'MANY_NAMESPACES':
            return `Simulate network policies for all deployments in ${scopeTextFormat.count} namespaces in cluster "${cluster}"`;
        case 'FEW_DEPLOYMENTS': {
            const deploymentText = pluralize('deployment', scopeTextFormat.deployments.length);
            const deployments = scopeTextFormat.deployments.join('", "');
            return `Simulate network policies for ${deploymentText} "${deployments}" in cluster "${cluster}"`;
        }
        case 'MANY_DEPLOYMENTS':
            return `Simulate network policies for ${scopeTextFormat.count} deployments in cluster "${cluster}"`;
        default:
            return ensureExhaustive(scopeTextFormat);
    }
}

export function getSimulationPanelResultsText(
    scopeHierarchy: Omit<NetworkScopeHierarchy, 'remainingQuery'>
): string {
    const scopeTextFormat = getScopeTextFormat(scopeHierarchy);
    const cluster = scopeHierarchy.cluster.name;

    switch (scopeTextFormat.type) {
        case 'CLUSTER_ONLY':
            return `Policies generated from the baseline of all deployments in cluster "${cluster}"`;
        case 'FEW_NAMESPACES': {
            const namespaceText = pluralize('namespace', scopeTextFormat.namespaces.length);
            const namespaces = scopeTextFormat.namespaces.join('", "');
            return `Policies generated from the baseline of all deployments in ${namespaceText} "${namespaces}" in cluster "${cluster}"`;
        }
        case 'MANY_NAMESPACES':
            return `Policies generated from the baseline of all deployments in ${scopeTextFormat.count} namespaces in cluster "${cluster}"`;
        case 'FEW_DEPLOYMENTS': {
            const deploymentText = pluralize('deployment', scopeTextFormat.deployments.length);
            const deployments = scopeTextFormat.deployments.join('", "');
            return `Policies generated from the baseline of ${deploymentText} "${deployments}" in cluster "${cluster}"`;
        }
        case 'MANY_DEPLOYMENTS':
            return `Policies generated from the baseline of ${scopeTextFormat.count} deployments in cluster "${cluster}"`;
        default:
            return ensureExhaustive(scopeTextFormat);
    }
}

type ScopeTextFormat =
    | { type: 'CLUSTER_ONLY' }
    | { type: 'FEW_NAMESPACES'; namespaces: string[] }
    | { type: 'MANY_NAMESPACES'; count: number }
    | { type: 'FEW_DEPLOYMENTS'; deployments: string[] }
    | { type: 'MANY_DEPLOYMENTS'; count: number };

/**
 * Given a selected scope, return the descriptive text format used to display longer text in the UI.
 * By encapsulating this logic we can make reuse and i16n easier in the future, as well as increase
 * readability of the text generators that uses this function.
 * @param scopeHierarchy The selected scope.
 * @param abbreviationLength The number of namespaces or deployments that can be displayed before
 *    the text is abbreviated.
 * @returns The descriptive text format used to display longer text in the UI.
 */
function getScopeTextFormat(
    scopeHierarchy: Omit<NetworkScopeHierarchy, 'remainingQuery'>,
    abbreviationLength = 3
): ScopeTextFormat {
    const { namespaces, deployments } = scopeHierarchy;
    // Cartesian product of selected namespaces and deployments
    const namespaceDeploymentPairs = namespaces.flatMap((n) => deployments.map((d) => `${n}/${d}`));

    if (namespaceDeploymentPairs.length === 0 && namespaces.length === 0) {
        return { type: 'CLUSTER_ONLY' };
    }
    if (namespaceDeploymentPairs.length === 0 && namespaces.length > abbreviationLength) {
        return { type: 'MANY_NAMESPACES', count: namespaces.length };
    }
    if (namespaceDeploymentPairs.length === 0 && namespaces.length <= abbreviationLength) {
        return { type: 'FEW_NAMESPACES', namespaces };
    }
    if (deployments.length > 0 && namespaceDeploymentPairs.length > abbreviationLength) {
        return { type: 'MANY_DEPLOYMENTS', count: namespaceDeploymentPairs.length };
    }
    if (deployments.length > 0) {
        return { type: 'FEW_DEPLOYMENTS', deployments: namespaceDeploymentPairs };
    }

    return { type: 'CLUSTER_ONLY' };
}
