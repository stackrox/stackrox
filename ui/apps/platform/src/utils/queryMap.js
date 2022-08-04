import entityTypes from 'constants/entityTypes';
import {
    SERVICE_ACCOUNTS_QUERY,
    SERVICE_ACCOUNT_QUERY,
    SERVICE_ACCOUNT_NAME,
} from 'queries/serviceAccount';
import { DEPLOYMENT_QUERY, DEPLOYMENTS_QUERY, DEPLOYMENT_NAME } from 'queries/deployment';
import { K8S_ROLES_QUERY, K8S_ROLE_QUERY, ROLE_NAME } from 'queries/role';
import { SECRET_QUERY, SECRETS_QUERY, SECRET_NAME } from 'queries/secret';
import { CLUSTER_QUERY, CLUSTERS_QUERY, CLUSTER_NAME } from 'queries/cluster';
import { CVE_NAME, IMAGE_CVE_NAME, NODE_CVE_NAME, CLUSTER_CVE_NAME } from 'queries/cve';
import { NAMESPACE_QUERY, NAMESPACES_QUERY, NAMESPACE_NAME } from 'queries/namespace';
import { POLICY_QUERY, POLICIES_QUERY, POLICY_NAME } from 'queries/policy';
import { CONTROL_QUERY, CONTROL_NAME } from 'queries/controls';
import { IMAGE_QUERY, IMAGES_QUERY, IMAGE_NAME } from 'queries/image';
import { NODES_QUERY, NODE_QUERY, NODE_NAME } from 'queries/node';
import { SUBJECTS_QUERY, SUBJECT_QUERY, SUBJECT_NAME } from 'queries/subject';

import { COMPONENT_NAME, NODE_COMPONENT_NAME, IMAGE_COMPONENT_NAME } from 'queries/components';

export const entityQueryMap = {
    [entityTypes.SERVICE_ACCOUNT]: SERVICE_ACCOUNT_QUERY,
    [entityTypes.SECRET]: SECRET_QUERY,
    [entityTypes.DEPLOYMENT]: DEPLOYMENT_QUERY,
    [entityTypes.CLUSTER]: CLUSTER_QUERY,
    [entityTypes.NAMESPACE]: NAMESPACE_QUERY,
    [entityTypes.ROLE]: K8S_ROLE_QUERY,
    [entityTypes.NODE]: NODE_QUERY,
    [entityTypes.CONTROL]: CONTROL_QUERY,
    [entityTypes.IMAGE]: IMAGE_QUERY,
    [entityTypes.POLICY]: POLICY_QUERY,
    [entityTypes.SUBJECT]: SUBJECT_QUERY,
};

export const entityListQueryMap = {
    [entityTypes.SERVICE_ACCOUNT]: SERVICE_ACCOUNTS_QUERY,
    [entityTypes.DEPLOYMENT]: DEPLOYMENTS_QUERY,
    [entityTypes.CLUSTER]: CLUSTERS_QUERY,
    [entityTypes.NAMESPACE]: NAMESPACES_QUERY,
    [entityTypes.ROLE]: K8S_ROLES_QUERY,
    [entityTypes.SECRET]: SECRETS_QUERY,
    [entityTypes.POLICY]: POLICIES_QUERY,
    [entityTypes.IMAGE]: IMAGES_QUERY,
    [entityTypes.NODE]: NODES_QUERY,
    [entityTypes.NAMESPACE]: NAMESPACES_QUERY,
    [entityTypes.SUBJECT]: SUBJECTS_QUERY,
};

export const entityNameQueryMap = {
    [entityTypes.CVE]: CVE_NAME,
    [entityTypes.IMAGE_CVE]: IMAGE_CVE_NAME,
    [entityTypes.NODE_CVE]: NODE_CVE_NAME,
    [entityTypes.CLUSTER_CVE]: CLUSTER_CVE_NAME,
    [entityTypes.SERVICE_ACCOUNT]: SERVICE_ACCOUNT_NAME,
    [entityTypes.SECRET]: SECRET_NAME,
    [entityTypes.DEPLOYMENT]: DEPLOYMENT_NAME,
    [entityTypes.CLUSTER]: CLUSTER_NAME,
    [entityTypes.NAMESPACE]: NAMESPACE_NAME,
    [entityTypes.ROLE]: ROLE_NAME,
    [entityTypes.NODE]: NODE_NAME,
    [entityTypes.CONTROL]: CONTROL_NAME,
    [entityTypes.IMAGE]: IMAGE_NAME,
    [entityTypes.POLICY]: POLICY_NAME,
    [entityTypes.SUBJECT]: SUBJECT_NAME,
    [entityTypes.COMPONENT]: COMPONENT_NAME,
    [entityTypes.NODE_COMPONENT]: NODE_COMPONENT_NAME,
    [entityTypes.IMAGE_COMPONENT]: IMAGE_COMPONENT_NAME,
};
