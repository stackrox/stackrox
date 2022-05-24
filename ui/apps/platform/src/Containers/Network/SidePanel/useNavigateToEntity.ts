import { useHistory } from 'react-router-dom';

import { EntityType } from 'Containers/Network/networkTypes';
import { nodeTypes } from 'constants/networkGraph';

export type NavigateToEntityHook = (entityId: string, type: EntityType) => void;

function useNavigateToEntity(): NavigateToEntityHook {
    const history = useHistory();
    return function onNavigateToEntityById(entityId: string, type: EntityType): void {
        if (type === nodeTypes.CIDR_BLOCK || type === nodeTypes.EXTERNAL_ENTITIES) {
            history.push(`/main/network/${entityId}/${type}${history.location.search as string}`);
        } else {
            history.push(`/main/network/${entityId}${history.location.search as string}`);
        }
    };
}

export default useNavigateToEntity;
