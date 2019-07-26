import entityTypes from 'constants/entityTypes';
import { SERVICE_ACCOUNT, SERVICE_ACCOUNTS } from 'queries/serviceAccount';
import {
    DEPLOYMENT_QUERY as DEPLOYMENT,
    DEPLOYMENTS_QUERY as DEPLOYMENTS
} from 'queries/deployment';
import { K8S_ROLES as ROLES, K8S_ROLE as ROLE } from 'queries/role';
import { SECRET, SECRETS } from 'queries/secret';
import { CLUSTER_QUERY as CLUSTER, CLUSTERS_QUERY as CLUSTERS } from 'queries/cluster';
import { NAMESPACE_QUERY as NAMESPACE, NAMESPACES_QUERY as NAMESPACES } from 'queries/namespace';
import { POLICY, POLICIES } from 'queries/policy';
import { CONTROL_QUERY as CONTROL } from 'queries/controls';
import { IMAGE, IMAGES } from 'queries/image';
import { NODES_QUERY as NODES, NODE_QUERY as NODE } from 'queries/node';
import { SUBJECTS_QUERY as SUBJECTS, SUBJECT_QUERY as SUBJECT } from 'queries/subject';

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
