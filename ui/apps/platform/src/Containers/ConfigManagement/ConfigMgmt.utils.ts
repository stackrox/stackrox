import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import type { GraphQLSortOption } from 'types/search';
import type { ConfigurationManagementEntityType } from 'utils/entityRelationships';

import { defaultClusterSort } from './List/ConfigManagementListClusters';
import { defaultDeploymentSort } from './List/ConfigManagementListDeployments';
import { defaultImageSort } from './List/ConfigManagementListImages';
import { defaultNamespaceSort } from './List/ConfigManagementListNamespaces';
import { defaultNodeSort } from './List/ConfigManagementListNodes';
import { defaultPolicySort } from './List/ConfigManagementListPolicies';
import { defaultRoleSort } from './List/ConfigManagementListRoles';
import { defaultSecretSort } from './List/ConfigManagementListSecrets';
import { defaultServiceAccountSort } from './List/ConfigManagementListServiceAccounts';
import { defaultSubjectSort } from './List/ConfigManagementListSubjects';

const defaultSortFieldMap: Record<ConfigurationManagementEntityType, GraphQLSortOption[]> = {
    CLUSTER: defaultClusterSort,
    CONTROL: [],
    DEPLOYMENT: defaultDeploymentSort,
    IMAGE: defaultImageSort,
    NAMESPACE: defaultNamespaceSort,
    NODE: defaultNodeSort,
    POLICY: defaultPolicySort,
    ROLE: defaultRoleSort,
    SECRET: defaultSecretSort,
    SERVICE_ACCOUNT: defaultServiceAccountSort,
    SUBJECT: defaultSubjectSort,
};

export function getConfigMgmtDefaultSort(
    entityListType: ConfigurationManagementEntityType
): GraphQLSortOption[] {
    const defaultSort = defaultSortFieldMap[entityListType];

    return defaultSort ?? [];
}

export function getConfigMgmtCountQuery(entityListType: ConfigurationManagementEntityType) {
    const parsedEntityListTypeCount = defaultCountKeyMap[entityListType];

    return !parsedEntityListTypeCount || entityListType === 'CONTROL' || entityListType === 'POLICY'
        ? ''
        : `count: ${parsedEntityListTypeCount}(query: $query)`;
}
