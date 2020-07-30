import isEqual from 'lodash/isEqual';
import capitalize from 'lodash/capitalize';

// Action types

export const types = (prefix) => ({
    SET_SEARCH_OPTIONS: `${prefix}/SET_SEARCH_OPTIONS`,
    SET_SEARCH_MODIFIERS: `${prefix}/SET_SEARCH_MODIFIERS`,
    SET_SEARCH_SUGGESTIONS: `${prefix}/SET_SEARCH_SUGGESTIONS`,
});

// Actions

export const getActions = (prefix) => {
    const actions = {};
    actions[`set${capitalize(prefix)}SearchOptions`] = (options) => ({
        type: `${prefix}/SET_SEARCH_OPTIONS`,
        options,
    });
    actions[`set${capitalize(prefix)}SearchModifiers`] = (modifiers) => ({
        type: `${prefix}/SET_SEARCH_MODIFIERS`,
        modifiers,
    });
    actions[`set${capitalize(prefix)}SearchSuggestions`] = (suggestions) => ({
        type: `${prefix}/SET_SEARCH_SUGGESTIONS`,
        suggestions,
    });
    return actions;
};

// Reducers

export const reducers = (prefix) => ({
    searchOptions: (state = [], action) => {
        if (action.type === `${prefix}/SET_SEARCH_OPTIONS`) {
            const { options } = action;
            return isEqual(options, state) ? state : options;
        }
        return state;
    },
    searchModifiers: (state = [], action) => {
        if (action.type === `${prefix}/SET_SEARCH_MODIFIERS`) {
            const { modifiers } = action;
            return isEqual(modifiers, state) ? state : modifiers;
        }
        return state;
    },
    searchSuggestions: (state = [], action) => {
        if (action.type === `${prefix}/SET_SEARCH_SUGGESTIONS`) {
            const { suggestions } = action;
            return isEqual(suggestions, state) ? state : suggestions;
        }
        return state;
    },
});

// Selectors

export const getSelectors = (prefix) => {
    const selectors = {};
    selectors[`get${capitalize(prefix)}SearchOptions`] = (state) => state.searchOptions;
    selectors[`get${capitalize(prefix)}SearchModifiers`] = (state) => state.searchModifiers;
    selectors[`get${capitalize(prefix)}SearchSuggestions`] = (state) => state.searchSuggestions;
    return selectors;
};
