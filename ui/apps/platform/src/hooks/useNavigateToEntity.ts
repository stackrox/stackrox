import { useHistory } from 'react-router-dom';

import { EntityType } from 'Containers/Network/networkTypes';
import { nodeTypes } from 'constants/networkGraph';

type NavigateHook = (entityId: string, type: EntityType) => void;

function useNavigateToEntity(): NavigateHook {
    const history = useHistory();
    return function onNavigateToEntityById(entityId: string, type: EntityType): void {
        if (type === nodeTypes.CIDR_BLOCK || type === nodeTypes.EXTERNAL_ENTITIES) {
            history.push(`/main/network/${entityId}/${type}`);
        } else {
            history.push(`/main/network/${entityId}`);
        }
    };
}

export default useNavigateToEntity;
