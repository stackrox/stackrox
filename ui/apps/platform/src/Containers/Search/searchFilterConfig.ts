import { alertAttributes } from 'Components/CompoundSearchFilter/attributes/alert';
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
import { policyAttributes } from 'Components/CompoundSearchFilter/attributes/policy';
import { policyCategoryAttributes } from 'Components/CompoundSearchFilter/attributes/policyCategory';
import { roleAttributes } from 'Components/CompoundSearchFilter/attributes/role';
import { roleBindingAttributes } from 'Components/CompoundSearchFilter/attributes/roleBinding';
import { secretAttributes } from 'Components/CompoundSearchFilter/attributes/secret';
import { serviceAccountAttributes } from 'Components/CompoundSearchFilter/attributes/serviceAccount';
import { subjectAttributes } from 'Components/CompoundSearchFilter/attributes/subject';
import type {
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

const entityForPolicy: CompoundSearchFilterEntity = {
    displayName: 'Policy',
    searchCategory: 'POLICIES',
    attributes: policyAttributes,
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

const entityForViolation: CompoundSearchFilterEntity = {
    displayName: 'Violations',
    searchCategory: 'ALERTS',
    attributes: alertAttributes,
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
