import { EntityType as alertEntityType } from 'Components/CompoundSearchFilter/attributes/alert';
import {
    clusterIdAttribute,
    clusterKubernetesVersionAttribute,
    clusterLabelAttribute,
    clusterNameAttribute,
    clusterPlatformTypeAttribute,
    clusterTypeAttribute,
} from 'Components/CompoundSearchFilter/attributes/cluster';
import { deploymentAttributes } from 'Components/CompoundSearchFilter/attributes/deployment';
import { EPSSProbability } from 'Components/CompoundSearchFilter/attributes/epssProbability';
import { imageAttributes } from 'Components/CompoundSearchFilter/attributes/image';
import { namespaceAttributes } from 'Components/CompoundSearchFilter/attributes/namespace';
import { nodeAttributes } from 'Components/CompoundSearchFilter/attributes/node';
import {
    Category as attributeForPolicyCategory,
    Description as attributeForPolicyDescription,
    Enforcement as attributeForPolicyEnforcement,
    LastUpdated as attributeForPolicyLastUpdated,
    LifecycleStage as attributeForPolicyLifecycleStage,
    Name as attributeForPolicyName,
    Severity as attributeForPolicySeverity,
    Status as attributeForPolicyStatus,
} from 'Components/CompoundSearchFilter/attributes/policy';
import { policyCategoryAttributes } from 'Components/CompoundSearchFilter/attributes/policyCategory';
import { roleAttributes } from 'Components/CompoundSearchFilter/attributes/role';
import { roleBindingAttributes } from 'Components/CompoundSearchFilter/attributes/roleBinding';
import { secretAttributes } from 'Components/CompoundSearchFilter/attributes/secret';
import { serviceAccountAttributes } from 'Components/CompoundSearchFilter/attributes/serviceAccount';
import { subjectAttributes } from 'Components/CompoundSearchFilter/attributes/subject';
import type {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterConfig,
    CompoundSearchFilterEntity,
} from 'Components/CompoundSearchFilter/types';
import type { SearchFieldLabel } from 'types/searchOptions';
import type { SearchFilter } from 'types/search';

/*
 * Global search spans every search result category. Each entity below maps to one result category
 * and exposes the legacy global-search fields for that category as typed, curated attributes
 * (feature parity with the old free-text global search bar).
 *
 * Home assignment: global search sends ONE flat query, and CompoundSearchFilterLabels renders chips
 * by flattening every entity's attributes, so a search term reused across entities renders a
 * duplicate chip (guarded by searchFilterConfig.test.ts). Each backend field therefore lives under
 * exactly one entity (its natural home); because the query is flat, the filter still matches results
 * in every category regardless of which entity hosts the term. Fields shared across categories
 * (CVE, Component, CVSS, ...) are homed on the entity they describe most directly.
 *
 * Source of truth: the Deployments field set is the authoritative options golden in
 * tests/options_test.go; the remaining per-entity fields come from the backend field list mirrored
 * in types/searchOptions.ts (the searchTerm type enforces that every value below is a real field).
 */

// Helpers to declare the many parity attributes concisely. Strings default to autocomplete (the
// backend suggests values, matching the legacy search bar); time fields use date-picker and numeric
// fields use condition-number.
function ac(
    displayName: string,
    filterChipLabel: string,
    searchTerm: SearchFieldLabel
): CompoundSearchFilterAttribute {
    return { displayName, filterChipLabel, searchTerm, inputType: 'autocomplete' };
}
function txt(
    displayName: string,
    filterChipLabel: string,
    searchTerm: SearchFieldLabel
): CompoundSearchFilterAttribute {
    return { displayName, filterChipLabel, searchTerm, inputType: 'text' };
}
function dt(
    displayName: string,
    filterChipLabel: string,
    searchTerm: SearchFieldLabel
): CompoundSearchFilterAttribute {
    return { displayName, filterChipLabel, searchTerm, inputType: 'date-picker' };
}
function num(
    displayName: string,
    filterChipLabel: string,
    searchTerm: SearchFieldLabel
): CompoundSearchFilterAttribute {
    return { displayName, filterChipLabel, searchTerm, inputType: 'condition-number' };
}

const entityForCluster: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: [
        clusterNameAttribute,
        clusterIdAttribute,
        clusterLabelAttribute,
        clusterTypeAttribute,
        clusterPlatformTypeAttribute,
        clusterKubernetesVersionAttribute,
    ],
};

const entityForDeployment: CompoundSearchFilterEntity = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: [
        ...deploymentAttributes,
        txt('Deployment type', 'Deployment type', 'Deployment Type'),
        ac('Pod label', 'Pod label', 'Pod Label'),
        ac('Image pull secret', 'Image pull secret', 'Image Pull Secret'),
        ac('Privileged', 'Privileged', 'Privileged'),
        ac('Read-only root filesystem', 'Read-only root filesystem', 'Read Only Root Filesystem'),
        ac('Add capabilities', 'Add capabilities', 'Add Capabilities'),
        ac('Drop capabilities', 'Drop capabilities', 'Drop Capabilities'),
        ac('Orchestrator component', 'Orchestrator component', 'Orchestrator Component'),
        ac('Platform component', 'Platform component', 'Platform Component'),
        num('CPU cores limit', 'CPU cores limit', 'CPU Cores Limit'),
        num('CPU cores request', 'CPU cores request', 'CPU Cores Request'),
        num('Memory limit (MB)', 'Memory limit (MB)', 'Memory Limit (MB)'),
        num('Memory request (MB)', 'Memory request (MB)', 'Memory Request (MB)'),
        ac('Environment key', 'Environment key', 'Environment Key'),
        ac('Environment value', 'Environment value', 'Environment Value'),
        ac('Environment source', 'Environment source', 'Environment Variable Source'),
        ac('Exposure level', 'Exposure level', 'Exposure Level'),
        ac('Max exposure level', 'Max exposure level', 'Max Exposure Level'),
        ac('Exposing service', 'Exposing service', 'Exposing Service'),
        num('Exposing service port', 'Exposing service port', 'Exposing Service Port'),
        num('Exposed node port', 'Exposed node port', 'Exposed Node Port'),
        num('Port', 'Port', 'Port'),
        ac('Port protocol', 'Port protocol', 'Port Protocol'),
        ac('External hostname', 'External hostname', 'External Hostname'),
        ac('External IP', 'External IP', 'External IP'),
        ac('Volume name', 'Volume name', 'Volume Name'),
        ac('Volume source', 'Volume source', 'Volume Source'),
        ac('Volume destination', 'Volume destination', 'Volume Destination'),
        ac('Volume type', 'Volume type', 'Volume Type'),
        ac('Volume read-only', 'Volume read-only', 'Volume ReadOnly'),
        ac('Process name', 'Process name', 'Process Name'),
        ac('Process path', 'Process path', 'Process Path'),
        ac('Process arguments', 'Process arguments', 'Process Arguments'),
        ac('Process UID', 'Process UID', 'Process UID'),
        dt('Process start time', 'Process start time', 'Process Container Start Time'),
    ],
};

// CVE and Component are single backend fields ('CVE', 'Component') shared by images and nodes, so
// they are homed once under Image (see the home-assignment note above); the flat query still matches
// node results. The remaining CVE/component attributes are likewise homed here.
const entityForImage: CompoundSearchFilterEntity = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: [
        ...imageAttributes,
        ac('Image ID', 'Image ID', 'Image ID'),
        ac('Image SHA', 'Image SHA', 'Image Sha'),
        ac('Remote', 'Image remote', 'Image Remote'),
        ac('Command', 'Image command', 'Image Command'),
        ac('Entrypoint', 'Image entrypoint', 'Image Entrypoint'),
        ac('User', 'Image user', 'Image User'),
        ac('Volumes', 'Image volumes', 'Image Volumes'),
        dt('Scan time', 'Image scan time', 'Image Scan Time'),
        num('CVE count', 'Image CVE count', 'Image CVE Count'),
        num('Top CVSS', 'Image top CVSS', 'Image Top CVSS'),
        ac('Dockerfile keyword', 'Dockerfile instruction keyword', 'Dockerfile Instruction Keyword'),
        ac('Dockerfile value', 'Dockerfile instruction value', 'Dockerfile Instruction Value'),
        ac('CVE', 'CVE', 'CVE'),
        ac('Component', 'Component', 'Component'),
        txt('Component version', 'Component version', 'Component Version'),
        num('CVSS', 'CVSS', 'CVSS'),
        num('NVD CVSS', 'NVD CVSS', 'NVD CVSS'),
        EPSSProbability,
        dt('CVE published', 'CVE published on', 'CVE Published On'),
        ac('CVE snoozed', 'CVE snoozed', 'CVE Snoozed'),
        ac('Vulnerability state', 'Vulnerability state', 'Vulnerability State'),
        ac('Fixable', 'Fixable', 'Fixable'),
        ac('Fixed by', 'Fixed by', 'Fixed By'),
        ac('Advisory name', 'Advisory name', 'Advisory Name'),
        ac('Advisory link', 'Advisory link', 'Advisory Link'),
    ],
};

const entityForNamespace: CompoundSearchFilterEntity = {
    displayName: 'Namespace',
    searchCategory: 'NAMESPACES',
    attributes: namespaceAttributes,
};

const entityForNode: CompoundSearchFilterEntity = {
    displayName: 'Node',
    searchCategory: 'NODES',
    attributes: [
        ...nodeAttributes,
        ac('ID', 'Node ID', 'Node ID'),
        ac('Container runtime', 'Container runtime', 'Container Runtime'),
        dt('Join time', 'Node join time', 'Node Join Time'),
        num('Top CVSS', 'Node top CVSS', 'Node Top CVSS'),
    ],
};

// Omit Violation time, Entity type, and Deployment status from the policy identity attributes: those
// share search terms with the Violation and Deployment entities and would render duplicate chips.
const entityForPolicy: CompoundSearchFilterEntity = {
    displayName: 'Policy',
    searchCategory: 'POLICIES',
    attributes: [
        attributeForPolicyName,
        ac('ID', 'Policy ID', 'Policy ID'),
        attributeForPolicyCategory,
        attributeForPolicySeverity,
        attributeForPolicyLifecycleStage,
        attributeForPolicyDescription,
        attributeForPolicyEnforcement,
        attributeForPolicyStatus,
        attributeForPolicyLastUpdated,
    ],
};

const entityForPolicyCategory: CompoundSearchFilterEntity = {
    displayName: 'Policy category',
    searchCategory: 'POLICY_CATEGORIES',
    attributes: [...policyCategoryAttributes, ac('ID', 'Policy category ID', 'Policy Category ID')],
};

const entityForRole: CompoundSearchFilterEntity = {
    displayName: 'Role',
    searchCategory: 'ROLES',
    attributes: [
        ...roleAttributes,
        ac('ID', 'Role ID', 'Role ID'),
        ac('Label', 'Role label', 'Role Label'),
        ac('Annotation', 'Role annotation', 'Role Annotation'),
        ac('Cluster role', 'Cluster role', 'Cluster Role'),
    ],
};

const entityForRoleBinding: CompoundSearchFilterEntity = {
    displayName: 'Role binding',
    searchCategory: 'ROLEBINDINGS',
    attributes: [
        ...roleBindingAttributes,
        ac('ID', 'Role binding ID', 'Role Binding ID'),
        ac('Label', 'Role binding label', 'Role Binding Label'),
        ac('Annotation', 'Role binding annotation', 'Role Binding Annotation'),
    ],
};

const entityForSecret: CompoundSearchFilterEntity = {
    displayName: 'Secret',
    searchCategory: 'SECRETS',
    attributes: [
        ...secretAttributes,
        ac('ID', 'Secret ID', 'Secret ID'),
        ac('Type', 'Secret type', 'Secret Type'),
        ac('Path', 'Secret path', 'Secret Path'),
        dt('Certificate expiration', 'Certificate expiration', 'Cert Expiration'),
        ac('Image pull secret registry', 'Image pull secret registry', 'Image Pull Secret Registry'),
    ],
};

const entityForServiceAccount: CompoundSearchFilterEntity = {
    displayName: 'Service account',
    searchCategory: 'SERVICE_ACCOUNTS',
    attributes: [
        ...serviceAccountAttributes,
        ac('Permission level', 'Service account permission level', 'Service Account Permission Level'),
        ac('Label', 'Service account label', 'Service Account Label'),
        ac('Annotation', 'Service account annotation', 'Service Account Annotation'),
    ],
};

const entityForSubject: CompoundSearchFilterEntity = {
    displayName: 'Users and groups',
    searchCategory: 'SUBJECTS',
    attributes: [...subjectAttributes, ac('Kind', 'Subject kind', 'Subject Kind')],
};

// Mirror the shared alert ViolationTime attribute but with a shorter display name: the
// "Violation" entity already supplies that context, so the attribute reads as just "Time".
const attributeForViolationTime: CompoundSearchFilterAttribute = {
    displayName: 'Time',
    filterChipLabel: 'Violation time',
    searchTerm: 'Violation Time',
    inputType: 'date-picker',
};

const entityForViolation: CompoundSearchFilterEntity = {
    displayName: 'Violation',
    searchCategory: 'ALERTS',
    attributes: [
        attributeForViolationTime,
        alertEntityType,
        ac('ID', 'Alert ID', 'Alert ID'),
        ac('Message', 'Violation', 'Violation'),
        ac('State', 'Violation state', 'Violation State'),
        ac('Resource', 'Resource', 'Resource'),
    ],
};

export const searchFilterConfig: CompoundSearchFilterConfig = [
    entityForCluster,
    entityForDeployment,
    entityForImage,
    entityForNamespace,
    entityForNode,
    entityForPolicy,
    entityForPolicyCategory,
    entityForRole,
    entityForRoleBinding,
    entityForSecret,
    entityForServiceAccount,
    entityForSubject,
    entityForViolation,
];

/*
 * Backwards-compatibility safety net for bookmarked or legacy global-search URLs.
 *
 * In normal operation this config covers every visible global search option, so parseQueryString
 * resolves a bookmarked key to one of two states: a term this config exposes (rendered as its typed
 * chip) or an unrecognized term (hidden/internal fields such as "Priority", or options removed over
 * time) which parseQueryString drops once the dynamic options load. The chip path below therefore
 * matters chiefly when the search-options metadata request is unavailable (still loading, or it
 * failed): parseQueryString then keeps every bookmarked key so the filter is not silently lost, and
 * without a matching config attribute CompoundSearchFilterLabels would render no chip, leaving an
 * invisible, unremovable filter.
 *
 * Build a plain-text attribute for each active filter key the config does not cover so it renders as
 * a removable chip. Matching is case-insensitive to mirror how configured chips resolve filter keys
 * (see getValueByCaseInsensitiveKey), which also prevents a duplicate chip for a case variant of a
 * configured term. Pass the result as CompoundSearchFilterLabels' attributesSeparateFromConfig.
 */
export function getUnconfiguredChipAttributes(
    searchFilter: SearchFilter,
    config: CompoundSearchFilterConfig
): CompoundSearchFilterAttribute[] {
    const configuredSearchTermsLower = new Set(
        config.flatMap((entity) =>
            entity.attributes.map((attribute) => attribute.searchTerm.toLowerCase())
        )
    );
    return Object.keys(searchFilter)
        .filter((key) => !configuredSearchTermsLower.has(key.toLowerCase()))
        .map(
            (key): CompoundSearchFilterAttribute => ({
                displayName: key,
                filterChipLabel: key,
                searchTerm: key as SearchFieldLabel,
                inputType: 'text',
            })
        );
}
