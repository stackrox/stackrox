function getStateArrayObject(type, entityId) {
    if (!type && !entityId) return null;
    const obj = { type };
    if (entityId) obj.id = entityId;

    return obj;
}

// Returns true if stack provided makes sense
function isStackValid(stack) {
    // TODO: Logic
    return !!stack;
}

// Checks state stack for overflow state and returns a valid trimmed version
function trimStack(stack) {
    // TODO: Logic

    // Navigate away if:
    // If there's no more room in the stack
    //
    // if the top entity is a parent of the entity before that then navigate away
    // List navigates to: Top single -> selected list
    // Entity navigates to : Entity page (maybe not)

    return stack;
}

function paramsToStateStack(params) {
    const {
        pageEntityListType,
        pageEntityType,
        pageEntityId,
        entityId1,
        entityId2,
        entityType1,
        entityType2,
        entityListType1,
        entityListType2
    } = params;

    const stateArray = [];
    if (!pageEntityListType && !pageEntityType) return stateArray;

    if (pageEntityListType) stateArray.push({ type: pageEntityListType });
    else stateArray.push({ type: pageEntityType, id: pageEntityId });

    const tab = entityListType1 ? { type: entityListType1 } : null;
    const entity1 = getStateArrayObject(
        entityType1 || entityListType1 || pageEntityListType,
        entityId1
    );
    const list = entityListType2 ? { type: entityListType2 } : null;
    const entity2 = getStateArrayObject(entityType2 || entityListType2, entityId2);
    // TODO: make this work
    if (tab) stateArray.push(tab);
    if (entity1) stateArray.push(entity1);
    if (list) stateArray.push(list);
    if (entity2) stateArray.push(entity2);

    if (!isStackValid)
        throw new Error('The supplied workflow state params produce an invalid state');

    return stateArray;
}

// Class that ensures the shape of a WorkflowState object
class WorkflowState {
    constructor(params) {
        if (!params.useCase) throw new Error('New WorkflowState must have a use case specified.');

        this.useCase = params.useCase;
        this.stateStack = paramsToStateStack(params);
    }

    setStack(newStack) {
        this.stateStack = newStack;
    }
}

export default class WorkflowStateMgr {
    constructor(workflowState, searchState) {
        this.workflowState = { ...workflowState };
        this.searchState = { ...searchState };
    }

    // Resets the current state based on minimal parameters
    reset(entityType, entityId, useCase) {
        const params = {
            useCase: useCase || this.workflowState.useCase
        };
        if (entityId) {
            params.pageEntityId = entityId;
            params.pageEntityType = entityType;
        } else {
            params.pageEntityListType = entityType;
        }

        this.workflowState = new WorkflowState(params);
        return this;
    }

    // Adds a list of entityType related to the current workflowState
    pushList(type) {
        const currentItem = this.stateStack.slice(-1)[0];
        if (!currentItem.id) {
            // replace the list type
            currentItem.type = type;
            return this;
        }

        this.workflowState.stateStack = trimStack([...this.stateStack, { type }]);
        return this;
    }

    // Selects an item in a list by Id
    pushListItem(id) {
        const currentItem = this.stateStack.slice(-1)[0];
        if (currentItem.id) {
            currentItem.id = id;
            return this;
        }
        // No trim necessary because you can always select an item from a list
        const newStack = [...this.stateStack, { type: currentItem.type, id }];
        this.workflowState.stateStack = newStack;

        return this;
    }

    // Shows an entity in relation to the top entity in the workflow
    pushRelatedEntity(type, id) {
        const currentItem = this.stateStack.slice(-1)[0];
        if (!currentItem.id)
            throw new Error(`Can't push related entity onto a list. Use pushListItem(id) instead.`);

        const newStack = trimStack([...this.stateStack, { type, id }]);
        this.workflowState.stateStack = newStack;

        return this;
    }

    // Goes back one level to the nearest valid state
    pop() {
        if (this.stateStack.length === 1)
            // A state stack has to have at least one item in it
            return this;

        this.workflowState.stateStack.pop();
        return this;
    }
}
