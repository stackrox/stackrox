import searchContexts from 'constants/searchContexts';

function getSidePanelEntity(workflowState, searchState) {
    const sidePanelSearch = searchState[searchContexts.sidePanel];
    if (workflowState.stateStack.length === 0) return {};

    const pageStack = workflowState.getPageStack();
    const sidePanelStateStack = workflowState.stateStack.slice(pageStack.length);

    // Calculate sidepanel entity props
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
        } else if (secondItem) {
            sidePanelEntityId = secondItem.entityId;
            sidePanelEntityType = secondItem.entityType;
            sidePanelListType = topItem.entityType;
        } else if (process.env.NODE_ENV === 'development') {
            throw new Error(
                `Neither topItem.entityId nor secondItem is defined in sidePanelStateStack.`
            );
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
