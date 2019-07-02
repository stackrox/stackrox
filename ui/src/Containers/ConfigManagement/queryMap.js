import entityTypes from 'constants/entityTypes';
import { SERVICE_ACCOUNT, SERVICE_ACCOUNTS } from 'queries/serviceAccount';
import {
    DEPLOYMENT_QUERY as DEPLOYMENT,
    DEPLOYMENTS_QUERY as DEPLOYMENTS
} from 'queries/deployment';
import { K8S_ROLES as ROLES, K8S_ROLE as ROLE } from 'queries/role';
import { SECRET, SECRETS } from 'queries/secret';
import { CLUSTER_QUERY as CLUSTER } from 'queries/cluster';
import { NAMESPACE_QUERY as NAMESPACE } from 'queries/namespace';
import { NODE_QUERY as NODE, NODES_QUERY as NODES } from 'queries/node';
import { CONTROL_QUERY as CONTROL } from 'queries/controls';
import { IMAGE, IMAGES } from 'queries/image';

export const entityQueryMap = {
    [entityTypes.SERVICE_ACCOUNT]: SERVICE_ACCOUNT,
    [entityTypes.SECRET]: SECRET,
    [entityTypes.DEPLOYMENT]: DEPLOYMENT,
    [entityTypes.CLUSTER]: CLUSTER,
    [entityTypes.NAMESPACE]: NAMESPACE,
    [entityTypes.ROLE]: ROLE,
    [entityTypes.NODE]: NODE,
    [entityTypes.CONTROL]: CONTROL,
    [entityTypes.IMAGE]: IMAGE
};

export const entityListQueryMap = {
    [entityTypes.SERVICE_ACCOUNT]: SERVICE_ACCOUNTS,
    [entityTypes.DEPLOYMENT]: DEPLOYMENTS,
    [entityTypes.ROLE]: ROLES,
    [entityTypes.SECRET]: SECRETS,
    [entityTypes.NODE]: NODES,
    [entityTypes.IMAGE]: IMAGES
};
