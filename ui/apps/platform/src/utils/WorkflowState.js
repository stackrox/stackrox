import cloneDeep from 'lodash/cloneDeep';

import entityRelationships from 'utils/entityRelationships';
import generateURL from 'utils/URLGenerator';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';

import WorkflowEntity from './WorkflowEntity';

// Returns true if stack provided makes sense
export function isStackValid(stack) {
    // stack is invalid when the stack is in one of three states:
    //
    // 1) entity -> (entity parent list) -> entity parent -> nav away
    // 2) entity -> (entity matches list) -> match entity -> nav away
    // 3) entity -> ... -> same entity (nav away)
    //
    // If you change the logic, then you will need to update a snapshot.

    const entityTypeMap = {};
    let entityTypeCount = 0;

    // if there is a duplicate element in the stack, it's always invalid
    for (let i = 0; i < stack.length; i += 1) {
        const { entityType, entityId } = stack[i];

        // checking if the entity type already exists in the map
        if (entityTypeMap[entityType]) {
            // it's a duplicate if an entity exists for that type
            if (entityTypeMap[entityType].hasEntity) {
                return false;
            }

            // it's a duplicate if the current entity is a list and a list exists for that type
            if (!entityId && entityTypeMap[entityType].hasList) {
                return false;
            }
        }

        if (!entityTypeMap[entityType]) {
            entityTypeCount += 1;
            entityTypeMap[entityType] = {};
        }
        if (entityId) {
            entityTypeMap[entityType].hasEntity = true;
        } else {
            entityTypeMap[entityType].hasList = true;
        }
    }

    // if the stack is smaller or equal to two entity types,
    // it is valid regardless of relationship type
    if (entityTypeCount <= 2) {
        return true;
    }

    for (let i = 1; i < stack.length; i += 1) {
        const { entityType: prevType } = stack[i - 1];
        const { entityType } = stack[i];

        if (prevType !== entityType) {
            // this checks if the current type on the stack is a parent of the previous type
            if (i !== stack.length - 1) {
                const isParent = entityRelationships.isContained(entityType, prevType);
                if (isParent) {
                    return false;
                }
            }

            // if prev entity type contains current entity type, match state doesn't matter and stack is valid
            const isContained = entityRelationships.isContained(prevType, entityType);
            if (!isContained) {
                // extended matches navigate away
                const isExtendedMatch = entityRelationships.isExtendedMatch(prevType, entityType);
                if (isExtendedMatch) {
                    return false;
                }

                // reflexive matches navigate away if it's not the last relationship on stack
                if (i !== stack.length - 1) {
                    const upMatch = entityRelationships.isPureMatch(prevType, entityType);
                    const downMatch = entityRelationships.isPureMatch(entityType, prevType);
                    if (upMatch && downMatch) {
                        return false;
                    }
                }
            }
        }
    }

    return true;
}

// Resets the current state based on minimal parameters
function baseStateStack(entityType, entityId) {
    return [new WorkflowEntity(entityType, entityId)];
}

// Returns skimmed stack for stack to navigate away to
function skimStack(stack) {
    if (stack.length < 2) {
        return stack;
    }

    const currentItem = stack.slice(-1)[0];
    // if the last item on the stack is an entity, return the entity
    if (currentItem.entityId) {
        return [currentItem];
    }
    // else the last item on the stack is a list, return the previous entity + related list
    return stack.slice(-2);
}

// Checks state stack for overflow state/invalid state and returns a valid skimmed version
function trimStack(stack) {
    // Navigate away if:
    // If there's no more "room" in the stack
    return isStackValid(stack) ? stack : skimStack(stack);
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

    getUseCase() {
        return this.useCase;
    }

    // Returns current entity (top of stack)
    getCurrentEntity() {
        if (!this.stateStack.length) {
            return null;
        }
        return this.stateStack.slice(-1)[0];
    }

    // Returns type of the current entity (top of stack)
    getCurrentEntityType() {
        const currentEntity = this.getCurrentEntity();

        if (!currentEntity) {
            return null;
        }

        return currentEntity.t;
    }

    getSingleAncestorOfType(entityType) {
        const ancestor = this.stateStack.find((level) => level.t === entityType && level.i);

        return ancestor ? new WorkflowEntity(ancestor.entityType, ancestor.entityId) : null;
    }

    // Returns base (first) entity of stack
    getBaseEntity() {
        if (!this.stateStack.length) {
            return null;
        }
        return this.stateStack[0];
    }

    getBaseEntityType() {
        const baseEntity = this.getBaseEntity();

        if (!baseEntity) {
            return '';
        }

        return baseEntity.t;
    }

    // Returns workflow entities related to page level
    getPageStack() {
        const { stateStack } = this;
        if (stateStack.length < 2) {
            return stateStack;
        }

        // list page or entity page with entity sidepanel
        if (!stateStack[0].entityId || (stateStack.length > 1 && stateStack[1].entityId)) {
            return stateStack.slice(0, 1);
        }

        // entity page with tab
        return stateStack.slice(0, 2);
    }

    // Gets selected table row (first side panel entity)
    getSelectedTableRow() {
        if (this.stateStack.length < 2 || !this.sidePanelActive) {
            return null;
        }
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

    // Returns skimmed stack version of WorkflowState to render into URL
    getSkimmedStack() {
        const { useCase, stateStack, search, sort, paging } = this;
        const newStateStack = skimStack(stateStack);
        const newSearch = search[searchParams.sidePanel]
            ? { [searchParams.page]: search[searchParams.sidePanel] }
            : null;
        const newSort = sort[sortParams.sidePanel]
            ? { [sortParams.page]: sort[sortParams.sidePanel] }
            : null;
        const newPaging = paging[pagingParams.sidePanel]
            ? { [pagingParams.page]: paging[pagingParams.sidePanel] }
            : null;
        return new WorkflowState(useCase, newStateStack, newSearch, newSort, newPaging);
    }

    getStateStack() {
        return this.stateStack;
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

    // Returns a cleared stack on current use case. Useful when building state from scratch.
    clear() {
        const newStateStack = [];
        const { useCase } = this;
        return new WorkflowState(useCase, newStateStack);
    }

    // sets the stateStack to base state when returning from side panel
    removeSidePanelParams() {
        const { useCase, search, sort, paging } = this;
        const newStateStack = this.getPageStack();
        const newSearch = search ? { [searchParams.page]: search[searchParams.page] } : null;
        const newSort = sort ? { [sortParams.page]: sort[sortParams.page] } : null;
        const newPaging = paging ? { [pagingParams.page]: paging[pagingParams.page] } : null;
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
        const trimmedStack = trimStack(newStateStack);
        const newPaging = trimmedStack.length === newStateStack.length ? paging : null;

        return new WorkflowState(useCase, trimStack(newStateStack), search, sort, newPaging);
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

        if (currentItem && !currentItem.entityId) {
            return this;
        }

        const newStateStack = trimStack([...stateStack, new WorkflowEntity(type, id)]);

        return new WorkflowState(useCase, newStateStack, search, sort, paging);
    }

    // Goes back one level to the nearest valid state
    pop() {
        if (this.stateStack.length === 1) {
            // A state stack has to have at least one item in it
            return this;
        }

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
            [param]: newProps,
        };
        return new WorkflowState(useCase, stateStack, newSearch, sort, paging);
    }

    setSort(sortProp) {
        const { useCase, stateStack, search, sort, paging, sidePanelActive } = this;
        const param = sidePanelActive ? sortParams.sidePanel : sortParams.page;

        const newSort = {
            ...sort,
            [param]: sortProp,
        };

        return new WorkflowState(useCase, stateStack, search, newSort, paging);
    }

    clearSearch() {
        const { useCase, stateStack, search, sort, paging, sidePanelActive } = this;
        const param = sidePanelActive ? searchParams.sidePanel : searchParams.page;

        const newSearch = {
            ...search,
            [param]: undefined,
        };

        return new WorkflowState(useCase, stateStack, newSearch, sort, paging);
    }

    clearSort() {
        const { useCase, stateStack, search, sort, paging, sidePanelActive } = this;
        const param = sidePanelActive ? sortParams.sidePanel : sortParams.page;

        const newSort = {
            ...sort,
            [param]: undefined,
        };

        return new WorkflowState(useCase, stateStack, search, newSort, paging);
    }

    setPage(pagingProp) {
        const { useCase, stateStack, search, sort, paging, sidePanelActive } = this;
        const param = sidePanelActive ? pagingParams.sidePanel : pagingParams.page;

        const newPaging = {
            ...paging,
            [param]: pagingProp,
        };
        return new WorkflowState(useCase, stateStack, search, sort, newPaging);
    }

    toUrl() {
        return generateURL(this);
    }

    getEntityContext() {
        return this.stateStack
            .filter((item) => !!item.entityId)
            .reduce((entityContext, item) => {
                return { ...entityContext, [item.entityType]: item.entityId };
            }, {});
    }

    // the following methods are helpers for very specific business logic
    /**
     * tests if the root of the state stack is the list of the entity type specified,
     *   with no child selected
     *
     * @param   {string}  entityType  the entityType constant for the entity list to check
     *
     * @return  {boolean}              true if the base of state stack is that entity list, false otherwise
     */
    isBaseList(entityType) {
        return this.stateStack[0] && this.stateStack[0].t === entityType && !this.stateStack[0].i;
    }

    /**
     * tests if the next to the last position on the state stack is a list of a given entity type
     *   (the part of the leaf entity)
     *
     * @param   {string}  entityType  the entityType constant to check
     *
     * @return  {boolean}              true if the preceding entity type is a list of the given entity type, false otherwise
     */
    isPreceding(entityType) {
        return (
            this.stateStack &&
            this.stateStack.length > 1 &&
            this.stateStack[this.stateStack.length - 2].t === entityType &&
            !!this.stateStack[this.stateStack.length - 2].i
        );
    }

    /**
     * tests if the next to the last position on the state stack is a single of a given entity type
     *   (the part of the leaf entity)
     *
     * @param   {string}  entityType  the entityType constant to check
     *
     * @return  {boolean}              true if the preceding entity type is a single of the given entity type, false otherwise
     */
    isPrecedingSingle(entityType) {
        return (
            this.stateStack &&
            this.stateStack.length > 1 &&
            this.stateStack[this.stateStack.length - 2].t === entityType &&
            this.stateStack[this.stateStack.length - 2].i
        );
    }

    /**
     * tests if the last position is single of a given entity type
     *   (the part of the leaf entity)
     *
     * @param   {string}  entityType  the entityType constant to check
     *
     * @return  {boolean}              true if last state is a single of the given entity type, false otherwise
     */
    isCurrentSingle(entityType) {
        return (
            this.stateStack &&
            this.stateStack.length > 0 &&
            this.stateStack[this.stateStack.length - 1].t === entityType &&
            this.stateStack[this.stateStack.length - 1].i
        );
    }
}
