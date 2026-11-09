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
import { imageAttributes } from 'Components/CompoundSearchFilter/attributes/image';
import { namespaceAttributes } from 'Components/CompoundSearchFilter/attributes/namespace';
import { nodeAttributes } from 'Components/CompoundSearchFilter/attributes/node';
import {
    Category as attributeForPolicyCategory,
    LifecycleStage as attributeForPolicyLifecycleStage,
    Name as attributeForPolicyName,
    Severity as attributeForPolicySeverity,
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

// Global search spans every search result category. Each entity below maps to one result
// category so the user can filter by curated, typed attributes. Entities reuse the shared
// attribute bundles where they exist; the thinner entities (policy category, role, role
// binding, secret, service account, subject) are modeled with a single Name attribute.

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
    attributes: deploymentAttributes,
};

const entityForImage: CompoundSearchFilterEntity = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: imageAttributes,
};

const entityForNamespace: CompoundSearchFilterEntity = {
    displayName: 'Namespace',
    searchCategory: 'NAMESPACES',
    attributes: namespaceAttributes,
};

const entityForNode: CompoundSearchFilterEntity = {
    displayName: 'Node',
    searchCategory: 'NODES',
    attributes: nodeAttributes,
};

// Reuse only the policy-identity attributes. The policyAttributes bundle also carries
// Violation time, Entity type, and Deployment status, but those share search terms with the
// Violations and Deployment entities and would render duplicate filter chips in global search.
const entityForPolicy: CompoundSearchFilterEntity = {
    displayName: 'Policy',
    searchCategory: 'POLICIES',
    attributes: [
        attributeForPolicyName,
        attributeForPolicyCategory,
        attributeForPolicySeverity,
        attributeForPolicyLifecycleStage,
    ],
};

const entityForPolicyCategory: CompoundSearchFilterEntity = {
    displayName: 'Policy category',
    searchCategory: 'POLICY_CATEGORIES',
    attributes: policyCategoryAttributes,
};

const entityForRole: CompoundSearchFilterEntity = {
    displayName: 'Role',
    searchCategory: 'ROLES',
    attributes: roleAttributes,
};

const entityForRoleBinding: CompoundSearchFilterEntity = {
    displayName: 'Role binding',
    searchCategory: 'ROLEBINDINGS',
    attributes: roleBindingAttributes,
};

const entityForSecret: CompoundSearchFilterEntity = {
    displayName: 'Secret',
    searchCategory: 'SECRETS',
    attributes: secretAttributes,
};

const entityForServiceAccount: CompoundSearchFilterEntity = {
    displayName: 'Service account',
    searchCategory: 'SERVICE_ACCOUNTS',
    attributes: serviceAccountAttributes,
};

const entityForSubject: CompoundSearchFilterEntity = {
    displayName: 'Subject',
    searchCategory: 'SUBJECTS',
    attributes: subjectAttributes,
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
    attributes: [attributeForViolationTime, alertEntityType],
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
