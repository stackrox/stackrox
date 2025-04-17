import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import { defaultClusterSort } from 'Containers/ConfigManagement/List/ConfigManagementListClusters';
import { defaultDeploymentSort } from 'Containers/ConfigManagement/List/ConfigManagementListDeployments';
import { defaultImageSort } from 'Containers/ConfigManagement/List/ConfigManagementListImages';
import { defaultNamespaceSort } from 'Containers/ConfigManagement/List/ConfigManagementListNamespaces';
import { defaultNodeSort } from 'Containers/ConfigManagement/List/ConfigManagementListNodes';
import { defaultPolicySort } from 'Containers/ConfigManagement/List/ConfigManagementListPolicies';
import { defaultRoleSort } from 'Containers/ConfigManagement/List/ConfigManagementListRoles';
import { defaultSecretSort } from 'Containers/ConfigManagement/List/ConfigManagementListSecrets';
import { defaultServiceAccountSort } from 'Containers/ConfigManagement/List/ConfigManagementListServiceAccounts';
import { defaultSubjectSort } from 'Containers/ConfigManagement/List/ConfigManagementListSubjects';
import { GraphQLSortOption } from 'types/search';
import { ConfigurationManagementEntityType } from 'utils/entityRelationships';

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
