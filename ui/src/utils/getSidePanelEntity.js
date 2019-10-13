import searchContexts from 'constants/searchContexts';

function getSidePanelEntity(stateStack, searchState) {
    const sidePanelSearch = searchState[searchContexts.sidePanel];
    if (!stateStack || stateStack.length === 0) return {};

    const baseEntity = stateStack[0];

    // Calculate sidepanel entity props
    const sidePanelStateStack = [...stateStack.slice(baseEntity.entityId ? 2 : 1)];
    const topItem = sidePanelStateStack.pop();
    const secondItem = sidePanelStateStack.pop();
    const sidePanelOpen = !!topItem;

    let sidePanelEntityId;
    let sidePanelEntityType;
    let sidePanelListType;
    if (sidePanelOpen) {
        if (topItem.entityId) {
            sidePanelEntityId = topItem.entityId;
            sidePanelEntityType = topItem.entityType;
        } else {
            sidePanelEntityId = secondItem.entityId;
            sidePanelEntityType = secondItem.entityType;
            sidePanelListType = topItem.entityType;
        }
    }

    return {
        sidePanelEntityId,
        sidePanelEntityType,
        sidePanelListType,
        sidePanelSearch
    };
}

export default getSidePanelEntity;
