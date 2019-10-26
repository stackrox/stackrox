import entityTypes from 'constants/entityTypes';
import { SERVICE_ACCOUNTS, SERVICE_ACCOUNT, SERVICE_ACCOUNT_NAME } from 'queries/serviceAccount';
import {
    DEPLOYMENT_QUERY as DEPLOYMENT,
    DEPLOYMENTS_QUERY as DEPLOYMENTS,
    DEPLOYMENT_NAME
} from 'queries/deployment';
import { K8S_ROLES as ROLES, K8S_ROLE as ROLE, ROLE_NAME } from 'queries/role';
import { SECRET, SECRETS, SECRET_NAME } from 'queries/secret';
import {
    CLUSTER_QUERY as CLUSTER,
    CLUSTERS_QUERY as CLUSTERS,
    CLUSTER_NAME
} from 'queries/cluster';
import { CVE_NAME } from 'queries/cve';
import {
    NAMESPACE_QUERY as NAMESPACE,
    NAMESPACES_QUERY as NAMESPACES,
    NAMESPACE_NAME
} from 'queries/namespace';
import { POLICY, POLICIES, POLICY_NAME } from 'queries/policy';
import { CONTROL_QUERY as CONTROL, CONTROL_NAME } from 'queries/controls';
import { IMAGE, IMAGES, IMAGE_NAME } from 'queries/image';
import { NODES_QUERY as NODES, NODE_QUERY as NODE, NODE_NAME } from 'queries/node';
import {
    SUBJECTS_QUERY as SUBJECTS,
    SUBJECT_QUERY as SUBJECT,
    SUBJECT_NAME
} from 'queries/subject';

import COMPONENT_NAME from 'queries/components';

export const entityQueryMap = {
    [entityTypes.SERVICE_ACCOUNT]: SERVICE_ACCOUNT,
    [entityTypes.SECRET]: SECRET,
    [entityTypes.DEPLOYMENT]: DEPLOYMENT,
    [entityTypes.CLUSTER]: CLUSTER,
    [entityTypes.NAMESPACE]: NAMESPACE,
    [entityTypes.ROLE]: ROLE,
    [entityTypes.NODE]: NODE,
    [entityTypes.CONTROL]: CONTROL,
    [entityTypes.IMAGE]: IMAGE,
    [entityTypes.POLICY]: POLICY,
    [entityTypes.SUBJECT]: SUBJECT
};

export const entityListQueryMap = {
    [entityTypes.SERVICE_ACCOUNT]: SERVICE_ACCOUNTS,
    [entityTypes.SECRET]: SECRETS,
    [entityTypes.DEPLOYMENT]: DEPLOYMENTS,
    [entityTypes.CLUSTER]: CLUSTERS,
    [entityTypes.NAMESPACE]: NAMESPACES,
    [entityTypes.ROLE]: ROLES,
    [entityTypes.SECRET]: SECRETS,
    [entityTypes.POLICY]: POLICIES,
    [entityTypes.IMAGE]: IMAGES,
    [entityTypes.NODE]: NODES,
    [entityTypes.NAMESPACE]: NAMESPACES,
    [entityTypes.POLICY]: POLICIES,
    [entityTypes.SUBJECT]: SUBJECTS
};

export const entityNameQueryMap = {
    [entityTypes.CVE]: CVE_NAME,
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
    [entityTypes.COMPONENT]: COMPONENT_NAME
};
