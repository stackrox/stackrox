import type { ConfigurationManagementEntityType } from 'utils/entityRelationships';
import { configManagementPath, urlEntityListTypes } from 'routePaths';

export function getConfigMgmtPathForEntitiesAndId(
    entityListType: ConfigurationManagementEntityType,
    id: string
) {
    return `${configManagementPath}/${urlEntityListTypes[entityListType]}/${id}`;
}
