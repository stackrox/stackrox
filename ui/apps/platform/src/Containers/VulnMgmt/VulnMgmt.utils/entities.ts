import useCaseTypes from 'constants/useCaseTypes';
import { VulnerabilityManagementEntityType } from 'utils/entityRelationships';
import { WorkflowState } from 'utils/WorkflowState';

const useCase = useCaseTypes.VULN_MANAGEMENT;

export function getVulnMgmtPathForEntitiesAndId(
    entityListType: VulnerabilityManagementEntityType,
    id: string
) {
    return new WorkflowState(useCase).pushList(entityListType).pushListItem(id).toUrl();
}
