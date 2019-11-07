import entityRelationships from 'modules/entityRelationships';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import cloneDeep from 'lodash/cloneDeep';
import generateURL from 'modules/URLGenerator';

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
    constructor(useCase, stateStack, search, sort, paging) {
        this.useCase = useCase;
        this.stateStack = cloneDeep(stateStack) || [];
        this.search = cloneDeep(search) || {};
        this.sort = cloneDeep(sort) || {};
        this.paging = cloneDeep(paging) || {};

        this.sidePanelActive = this.getPageStack().length !== this.stateStack.length;

        Object.freeze(this);
        Object.freeze(this.search);
        Object.freeze(this.stateStack);
        Object.freeze(this.sort);
        Object.freeze(this.paging);
    }

    clone() {
        const { useCase, stateStack, search, sort, paging } = this;
        return new WorkflowState(useCase, stateStack, search, sort, paging);
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

    // Gets selected table row (first side panel entity)
    getSelectedTableRow() {
        if (this.stateStack.length < 2 || !this.sidePanelActive) return null;
        return this.stateStack.slice(1, 2)[0];
    }

    getCurrentSearchState() {
        const param = this.sidePanelActive ? searchParams.sidePanel : searchParams.page;
        return this.search[param] || {};
    }

    getCurrentSortState() {
        const param = this.sidePanelActive ? sortParams.sidePanel : sortParams.page;
        return this.sort[param] || {};
    }

    getCurrentPagingState() {
        const param = this.sidePanelActive ? pagingParams.sidePanel : pagingParams.page;
        return this.paging[param] || {};
    }

    // Resets the current state based on minimal parameters
    reset(useCase, entityType, entityId, search, sort, paging) {
        const newUseCase = useCase || this.useCase;
        const newStateStack = baseStateStack(entityType, entityId);
        return new WorkflowState(newUseCase, newStateStack, search, sort, paging);
    }

    resetPage(type, id) {
        const newStateStack = [new WorkflowEntity(type, id)];

        const { useCase } = this;
        return new WorkflowState(useCase, newStateStack);
    }

    // sets the stateStack to base state when returning from side panel
    removeSidePanelParams() {
        const { useCase, stateStack, search, sort, paging } = this;
        const baseEntity = this.getBaseEntity();
        const entityTab = stateStack[1] && !stateStack[1].entityId;
        const newStateStack =
            baseEntity.entityId && entityTab ? stateStack.slice(0, 2) : [baseEntity];
        const newSearch = search ? { [searchParams.page]: search[searchParams.page] } : null;
        const newSort = sort ? { [searchParams.page]: sort[searchParams.page] } : null;
        const newPaging = paging ? { [searchParams.page]: paging[searchParams.page] } : null;
        return new WorkflowState(useCase, newStateStack, newSearch, newSort, newPaging);
    }

    // sets statestack to only the first item
    base() {
        const { useCase, stateStack } = this;
        return new WorkflowState(useCase, stateStack.slice(0, 1));
    }

    // Adds a list of entityType related to the current workflowState
    pushList(type) {
        const { useCase, stateStack, search, sort, paging } = this;
        const newItem = new WorkflowEntity(type);
        const currentItem = this.getCurrentEntity();

        // Slice an item off the end of the stack if this push should result in a replacement (e.g. clicking on tabs)
        const newStateStack =
            currentItem && currentItem.entityType && !currentItem.entityId
                ? stateStack.slice(0, -1)
                : [...stateStack];
        newStateStack.push(newItem);

        return new WorkflowState(useCase, trimStack(newStateStack), search, sort, paging);
    }

    // Selects an item in a list by Id
    pushListItem(id) {
        const { useCase, stateStack, search, sort, paging } = this;
        const currentItem = this.getCurrentEntity();
        const newItem = new WorkflowEntity(currentItem.entityType, id);
        // Slice an item off the end of the stack if this push should result in a replacement (e.g. clicking on multiple list items)
        const newStateStack = currentItem.entityId ? stateStack.slice(0, -1) : [...stateStack];
        newStateStack.push(newItem);

        return new WorkflowState(useCase, newStateStack, search, sort, paging);
    }

    // Shows an entity in relation to the top entity in the workflow
    pushRelatedEntity(type, id) {
        const { useCase, stateStack, search, sort, paging } = this;
        const currentItem = stateStack.slice(-1)[0];

        if (currentItem && !currentItem.entityId) return this;

        const newStateStack = trimStack([...stateStack, new WorkflowEntity(type, id)]);

        return new WorkflowState(useCase, newStateStack, search, sort, paging);
    }

    // Goes back one level to the nearest valid state
    pop() {
        if (this.stateStack.length === 1)
            // A state stack has to have at least one item in it
            return this;

        const { useCase, stateStack, search, sort, paging } = this;

        return new WorkflowState(
            useCase,
            stateStack.slice(0, stateStack.length - 1),
            search,
            sort,
            paging
        );
    }

    setSearch(newProps) {
        const { useCase, stateStack, search, sort, paging, sidePanelActive } = this;
        const param = sidePanelActive ? searchParams.sidePanel : searchParams.page;

        const newSearch = {
            ...search,
            [param]: newProps
        };
        return new WorkflowState(useCase, stateStack, newSearch, sort, paging);
    }

    setSort(sortProp) {
        const { useCase, stateStack, search, sort, paging, sidePanelActive } = this;
        const param = sidePanelActive ? sortParams.sidePanel : sortParams.page;

        const newSort = {
            ...sort,
            [param]: sortProp
        };
        return new WorkflowState(useCase, stateStack, search, newSort, paging);
    }

    setPage(pagingProp) {
        const { useCase, stateStack, search, sort, paging, sidePanelActive } = this;
        const param = sidePanelActive ? pagingParams.sidePanel : pagingParams.page;

        const newPaging = {
            ...paging,
            [param]: pagingProp
        };
        return new WorkflowState(useCase, stateStack, search, sort, newPaging);
    }

    toUrl() {
        return generateURL(this);
    }

    getEntityContext() {
        return this.stateStack
            .filter(item => !!item.entityId)
            .reduce((entityContext, item) => {
                return { ...entityContext, [item.entityType]: item.entityId };
            }, {});
    }
}
