import entityTypes from 'constants/entityTypes';
import { SERVICE_ACCOUNT_NAME } from 'queries/serviceAccount';
import { DEPLOYMENT_NAME } from 'queries/deployment';
import { ROLE_NAME } from 'queries/role';
import { SECRET_NAME } from 'queries/secret';
import { CLUSTER_NAME } from 'queries/cluster';
import { CVE_NAME, IMAGE_CVE_NAME, NODE_CVE_NAME, CLUSTER_CVE_NAME } from 'queries/cve';
import { NAMESPACE_NAME } from 'queries/namespace';
import { POLICY_NAME } from 'queries/policy';
import { CONTROL_NAME } from 'queries/controls';
import { IMAGE_NAME } from 'queries/image';
import { NODE_NAME } from 'queries/node';
import { SUBJECT_NAME } from 'queries/subject';

import { COMPONENT_NAME, NODE_COMPONENT_NAME, IMAGE_COMPONENT_NAME } from 'queries/components';

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
