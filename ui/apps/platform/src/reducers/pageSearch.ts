import isEqual from 'lodash/isEqual';
import capitalize from 'lodash/capitalize';
import { SearchEntry } from 'types/search';

export type SearchState = {
    searchOptions: SearchEntry[];
    searchModifiers: SearchEntry[];
    searchSuggestions: SearchEntry[];
};

// Action types

type SetSearchOptionsActionType<T extends string> = `${T}/SET_SEARCH_OPTIONS`;
type SetSearchModifiersActionType<T extends string> = `${T}/SET_SEARCH_MODIFIERS`;
type SetSearchSuggestionsActionType<T extends string> = `${T}/SET_SEARCH_SUGGESTIONS`;

type SearchActionTypes<T extends string> = {
    SET_SEARCH_OPTIONS: SetSearchOptionsActionType<T>;
    SET_SEARCH_MODIFIERS: SetSearchModifiersActionType<T>;
    SET_SEARCH_SUGGESTIONS: SetSearchSuggestionsActionType<T>;
};

export function types<EntityPrefix extends string>(
    prefix: EntityPrefix
): SearchActionTypes<EntityPrefix> {
    return {
        SET_SEARCH_OPTIONS: `${prefix}/SET_SEARCH_OPTIONS`,
        SET_SEARCH_MODIFIERS: `${prefix}/SET_SEARCH_MODIFIERS`,
        SET_SEARCH_SUGGESTIONS: `${prefix}/SET_SEARCH_SUGGESTIONS`,
    } as SearchActionTypes<EntityPrefix>;
}

// Actions

type SetSearchOptionsAction<T extends string> = {
    type: SetSearchOptionsActionType<T>;
    options: SearchEntry[];
};
type SetSearchModifiersAction<T extends string> = {
    type: SetSearchModifiersActionType<T>;
    modifiers: SearchEntry[];
};
type SetSearchSuggestionsAction<T extends string> = {
    type: SetSearchSuggestionsActionType<T>;
    suggestions: SearchEntry[];
};

type SearchActions<T extends string> = {
    [prop in `set${Capitalize<T>}SearchOptions`]: (
        options: SearchEntry[]
    ) => SetSearchOptionsAction<T>;
} & {
    [prop in `set${Capitalize<T>}SearchModifiers`]: (
        modifiers: SearchEntry[]
    ) => SetSearchModifiersAction<T>;
} & {
    [prop in `set${Capitalize<T>}SearchSuggestions`]: (
        suggestions: SearchEntry[]
    ) => SetSearchSuggestionsAction<T>;
};

export function getActions<EntityPrefix extends string>(
    prefix: EntityPrefix
): SearchActions<EntityPrefix> {
    return {
        [`set${capitalize(prefix)}SearchOptions`]: (options: SearchEntry[]) => ({
            type: `${prefix}/SET_SEARCH_OPTIONS`,
            options,
        }),
        [`set${capitalize(prefix)}SearchModifiers`]: (modifiers: SearchEntry[]) => ({
            type: `${prefix}/SET_SEARCH_MODIFIERS`,
            modifiers,
        }),
        [`set${capitalize(prefix)}SearchSuggestions`]: (suggestions: SearchEntry[]) => ({
            type: `${prefix}/SET_SEARCH_SUGGESTIONS`,
            suggestions,
        }),
    } as SearchActions<EntityPrefix>;
}

// Reducers

type OneOfSearchActions<T extends string> =
    | SetSearchOptionsAction<T>
    | SetSearchModifiersAction<T>
    | SetSearchSuggestionsAction<T>;

type SearchReducer<T extends string> = (
    state: SearchEntry[],
    action: OneOfSearchActions<T>
) => SearchEntry[];

type Reducers<T extends string> = {
    searchOptions: SearchReducer<T>;
    searchModifiers: SearchReducer<T>;
    searchSuggestions: SearchReducer<T>;
};

export function reducers<EntityPrefix extends string>(
    prefix: EntityPrefix
): Reducers<EntityPrefix> {
    const searchOptions: SearchReducer<EntityPrefix> = (state = [], action) => {
        if (action.type === `${prefix}/SET_SEARCH_OPTIONS`) {
            const { options } = action as SetSearchOptionsAction<EntityPrefix>;
            return isEqual(options, state) ? state : options;
        }
        return state;
    };
    const searchModifiers: SearchReducer<EntityPrefix> = (state = [], action) => {
        if (action.type === `${prefix}/SET_SEARCH_MODIFIERS`) {
            const { modifiers } = action as SetSearchModifiersAction<EntityPrefix>;
            return isEqual(modifiers, state) ? state : modifiers;
        }
        return state;
    };
    const searchSuggestions: SearchReducer<EntityPrefix> = (state = [], action) => {
        if (action.type === `${prefix}/SET_SEARCH_SUGGESTIONS`) {
            const { suggestions } = action as SetSearchSuggestionsAction<EntityPrefix>;
            return isEqual(suggestions, state) ? state : suggestions;
        }
        return state;
    };
    return {
        searchOptions,
        searchModifiers,
        searchSuggestions,
    };
}

// Selectors

type SearchSelectorNames<T extends string> =
    | `get${Capitalize<T>}SearchOptions`
    | `get${Capitalize<T>}SearchModifiers`
    | `get${Capitalize<T>}SearchSuggestions`;

type SearchSelectors<T extends string> = {
    [prop in SearchSelectorNames<T>]: (state: SearchState) => SearchEntry[];
};

export function getSelectors<EntityPrefix extends string>(
    prefix: EntityPrefix
): SearchSelectors<EntityPrefix> {
    return {
        [`get${capitalize(prefix)}SearchOptions`]: (state: SearchState) => state.searchOptions,
        [`get${capitalize(prefix)}SearchModifiers`]: (state: SearchState) => state.searchModifiers,
        [`get${capitalize(prefix)}SearchSuggestions`]: (state: SearchState) =>
            state.searchSuggestions,
    } as SearchSelectors<EntityPrefix>;
}
