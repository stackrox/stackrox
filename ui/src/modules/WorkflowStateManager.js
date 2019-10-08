import entityRelationships from 'modules/entityRelationships';

// An item in the workflow stack
export class WorkflowEntity {
    constructor(entityType, entityId) {
        if (entityType) {
            this.entityType = entityType;
        }
        if (entityId) {
            this.i = entityId;
        }
    }

    get entityType() {
        return this.t;
    }

    get entityId() {
        return this.i;
    }

    set entityType(entityType) {
        this.t = entityType;
    }

    set entityId(entityId) {
        this.i = entityId;
    }
}

// Returns true if stack provided makes sense
export function isStackValid(stack) {
    if (stack.length < 2) return true;

    // stack is invalid when the stack is in one of three states:
    //
    // 1) entity -> (entity parent list) -> entity parent -> nav away
    // 2) entity -> (entity matches list) -> match entity -> nav away
    // 3) entity -> (entity contains-inferred list) -> contains-inferred entity -> nav away

    let isParentState;
    let isMatchState;
    let isInferredState;

    stack.forEach((entity, i) => {
        const { entityType } = entity;
        if (i > 0 && i !== stack.length - 1) {
            const { entityType: prevType } = stack[i - 1];
            if (!isParentState) {
                isParentState = entityRelationships.isParent(entityType, prevType);
            }
            if (!isMatchState) {
                isMatchState = entityRelationships.isMatch(entityType, prevType);
            }
            if (!isInferredState) {
                isInferredState = entityRelationships.isContainedInferred(prevType, entityType);
            }
        }
        return false;
    });
    return !isParentState && !isMatchState && !isInferredState;
}

// Resets the current state based on minimal parameters
function baseStateStack(entityType, entityId) {
    return [new WorkflowEntity(entityType, entityId)];
}

// Checks state stack for overflow state/invalid state and returns a valid trimmed version
function trimStack(stack) {
    // Navigate away if:
    // If there's no more "room" in the stack

    // if the top entity is a parent of the entity before that then navigate away
    // List navigates to: Top single -> selected list
    // Entity navigates to : Entity page (maybe not)
    if (isStackValid(stack)) return stack;
    const { entityType: lastItemType, entityId: lastItemId } = stack.slice(-1)[0];
    if (!lastItemId) {
        const { entityType, entityId } = stack.slice(-2)[0];
        return [...baseStateStack(entityType, entityId), new WorkflowEntity(lastItemType)];
    }
    return baseStateStack(lastItemType, lastItemId);
}

/**
 * Summary: Class that ensures the shape of a WorkflowState object
 * {
 *   useCase: 'text',
 *   stateStack: [{t: 'entityType', i: 'entityId'},{t: 'entityType', i: 'entityId'}]
 * }
 */
export class WorkflowState {
    constructor(useCase, stateStack) {
        this.useCase = useCase;
        this.stateStack = stateStack || [];
    }

    // Returns current entity (top of stack)
    getCurrentEntity() {
        if (!this.stateStack.length) return null;
        return this.stateStack.slice(-1)[0];
    }

    // Returns base (first) entity of stack
    getBaseEntity() {
        if (!this.stateStack.length) return null;
        return this.stateStack[0];
    }
}

export default class WorkflowStateMgr {
    constructor(workflowState, searchState) {
        if (workflowState) {
            const { useCase, stateStack } = workflowState;
            this.workflowState = new WorkflowState(useCase, stateStack);
        } else {
            this.workflowState = new WorkflowState();
        }
        this.searchState = { ...searchState };
    }

    // Resets the current state based on minimal parameters
    reset(useCase, entityType, entityId) {
        const newUseCase = useCase || this.workflowState.useCase;
        const newStateStack = baseStateStack(entityType, entityId);

        this.workflowState = new WorkflowState(newUseCase, newStateStack);
        return this;
    }

    // sets the stateStack to base state when returning from side panel
    base() {
        const { useCase, stateStack } = this.workflowState;
        const baseEntity = this.workflowState.getBaseEntity();
        const newStateStack = baseEntity.entityId ? stateStack.slice(0, 2) : [baseEntity];
        this.workflowState = new WorkflowState(useCase, newStateStack);
        return this;
    }

    // Adds a list of entityType related to the current workflowState
    pushList(type) {
        const listState = new WorkflowEntity(type);

        // if coming from dashboard
        if (!this.workflowState.stateStack.length) {
            this.workflowState.stateStack = [listState];
            return this;
        }

        const currentItem = this.workflowState.stateStack.slice(-1)[0];
        if (currentItem.entityType && !currentItem.entityId) {
            // replace the list type
            currentItem.entityType = type;
            return this;
        }

        this.workflowState.stateStack = trimStack([...this.workflowState.stateStack, listState]);
        return this;
    }

    // Selects an item in a list by Id
    pushListItem(id) {
        const currentItem = this.workflowState.stateStack.slice(-1)[0];
        // this shouldn't happen since the panel closes on clicking out, but just in case
        if (currentItem.entityId) {
            currentItem.entityId = id;
            return this;
        }
        this.workflowState.stateStack.push(new WorkflowEntity(currentItem.entityType, id));
        return this;
    }

    // Shows an entity in relation to the top entity in the workflow
    pushRelatedEntity(type, id) {
        const currentItem = this.workflowState.stateStack.slice(-1)[0];
        if (!currentItem.entityId)
            throw new Error(`Can't push related entity onto a list. Use pushListItem(id) instead.`);

        const newStack = trimStack([
            ...this.workflowState.stateStack,
            new WorkflowEntity(type, id)
        ]);
        this.workflowState.stateStack = newStack;

        return this;
    }

    // Goes back one level to the nearest valid state
    pop() {
        if (this.workflowState.stateStack.length === 1)
            // A state stack has to have at least one item in it
            return this;

        this.workflowState.stateStack.pop();
        return this;
    }
}
