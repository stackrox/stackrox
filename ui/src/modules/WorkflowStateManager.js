import entityRelationships from 'modules/entityRelationships';
import { cloneDeep } from 'lodash';
import searchContexts from 'constants/searchContexts';

// An item in the workflow stack
export class WorkflowEntity {
    constructor(entityType, entityId) {
        if (entityType) {
            this.t = entityType;
        }
        if (entityId) {
            this.i = entityId;
        }
        Object.freeze(this);
    }

    get entityType() {
        return this.t;
    }

    get entityId() {
        return this.i;
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
    constructor(useCase, stateStack, search) {
        this.useCase = useCase;
        this.stateStack = cloneDeep(stateStack) || [];
        this.search = search || {};

        Object.freeze(this);
        Object.freeze(search);
        Object.freeze(stateStack);
        Object.freeze(useCase);
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

    // Gets workflow entities related to page level
    getPageStack() {
        const { stateStack } = this;
        if (stateStack.length < 2) return stateStack;

        // list page or entity page with entity sidepanel
        if (!stateStack[0].entityId || (stateStack.length > 1 && stateStack[1].entityId))
            return stateStack.slice(0, 1);

        // entity page with tab
        return stateStack.slice(0, 2);
    }

    getCurrentSearchContext() {
        return this.getPageStack().length === this.stateStack.length
            ? searchContexts.page
            : searchContexts.sidePanel;
    }

    getCurrentSearchState() {
        return this.search[this.getCurrentSearchContext()] || {};
    }
}

export default class WorkflowStateMgr {
    constructor(workflowState) {
        if (workflowState) {
            const { useCase, stateStack, search } = workflowState;
            this.workflowState = new WorkflowState(useCase, stateStack, search);
        } else {
            this.workflowState = new WorkflowState();
        }
    }

    // Resets the current state based on minimal parameters
    reset(useCase, entityType, entityId, search) {
        const newUseCase = useCase || this.workflowState.useCase;
        const newStateStack = baseStateStack(entityType, entityId);
        const newSearch = search || this.search;
        this.workflowState = new WorkflowState(newUseCase, newStateStack, newSearch);
        return this;
    }

    // sets the stateStack to base state when returning from side panel
    removeSidePanelParams() {
        const { useCase, stateStack, search } = this.workflowState;
        const baseEntity = this.workflowState.getBaseEntity();
        const newStateStack = baseEntity.entityId ? stateStack.slice(0, 2) : [baseEntity];
        const newSearch = { [searchContexts.page]: search[searchContexts.page] };
        this.workflowState = new WorkflowState(useCase, newStateStack, newSearch);
        return this;
    }

    // sets statestack to only the first item
    base() {
        const { useCase, stateStack, search } = this.workflowState;
        this.workflowState = new WorkflowState(useCase, stateStack.slice(0, 1), search);
        return this;
    }

    // Adds a list of entityType related to the current workflowState
    pushList(type) {
        const { useCase, stateStack, search } = this.workflowState;
        const newItem = new WorkflowEntity(type);
        const currentItem = this.workflowState.getCurrentEntity();

        // Slice an item off the end of the stack if this push should result in a replacement (e.g. clicking on tabs)
        const newStateStack =
            currentItem && currentItem.entityType && !currentItem.entityId
                ? stateStack.slice(0, -1)
                : stateStack;
        newStateStack.push(newItem);

        this.workflowState = new WorkflowState(useCase, trimStack(newStateStack), search);

        return this;
    }

    // Selects an item in a list by Id
    pushListItem(id) {
        const { useCase, stateStack, search } = this.workflowState;
        const currentItem = this.workflowState.getCurrentEntity();
        const newItem = new WorkflowEntity(currentItem.entityType, id);
        // Slice an item off the end of the stack if this push should result in a replacement (e.g. clicking on multiple list items)
        const newStateStack = currentItem.entityId ? stateStack.slice(0, -1) : stateStack;
        newStateStack.push(newItem);

        this.workflowState = new WorkflowState(useCase, newStateStack, search);
        return this;
    }

    // Shows an entity in relation to the top entity in the workflow
    pushRelatedEntity(type, id) {
        const { useCase, stateStack, search } = this.workflowState;
        const currentItem = stateStack.slice(-1)[0];
        if (!currentItem.entityId)
            throw new Error(`Can't push related entity onto a list. Use pushListItem(id) instead.`);

        const newStateStack = trimStack([...stateStack, new WorkflowEntity(type, id)]);

        this.workflowState = new WorkflowState(useCase, newStateStack, search);

        return this;
    }

    // Goes back one level to the nearest valid state
    pop() {
        if (this.workflowState.stateStack.length === 1)
            // A state stack has to have at least one item in it
            return this;

        const { useCase, stateStack, search } = this.workflowState;

        this.workflowState = new WorkflowState(
            useCase,
            stateStack.slice(0, stateStack.length - 1),
            search
        );
        return this;
    }

    setSearch(newProps) {
        const { useCase, stateStack, search } = this.workflowState;
        const newSearch = {
            ...search,
            [this.workflowState.getCurrentSearchContext()]: newProps
        };
        this.workflowState = new WorkflowState(useCase, stateStack, newSearch);
    }
}
