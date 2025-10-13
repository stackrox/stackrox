import React, { Component } from 'react';
import PropTypes from 'prop-types';
import findLastIndex from 'lodash/findLastIndex';

import { Creatable } from 'Components/ReactSelect';
import {
    placeholderCreator,
    Option,
    ValueContainer,
    MultiValue,
    noOptionsMessage,
    createOptionPosition,
    inputMatchesTopOption,
    removeValuesForKey,
} from 'Components/URLSearchInputWithAutocomplete';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

export const searchInputPropTypes = {
    className: PropTypes.string,
    placeholder: PropTypes.string.isRequired,
    searchOptions: PropTypes.arrayOf(PropTypes.object),
    searchModifiers: PropTypes.arrayOf(PropTypes.object),
    setSearchOptions: PropTypes.func.isRequired,
    onSearch: PropTypes.func,
    isGlobal: PropTypes.bool,
    defaultOption: PropTypes.shape({
        value: PropTypes.string,
        label: PropTypes.string,
        category: PropTypes.string,
    }),
    autoCompleteResults: PropTypes.arrayOf(PropTypes.string),
    sendAutoCompleteRequest: PropTypes.func,
    clearAutoComplete: PropTypes.func,
    autoCompleteCategories: PropTypes.arrayOf(PropTypes.string),
    setAllSearchOptions: PropTypes.func,
    isDisabled: PropTypes.bool,
    prependAutocompleteQuery: PropTypes.arrayOf(
        PropTypes.shape({
            value: PropTypes.string,
            category: PropTypes.string,
        })
    ),
};

export const searchInputDefaultProps = {
    className: '',
    searchOptions: [],
    searchModifiers: [],
    onSearch: null,
    isGlobal: false,
    defaultOption: null,
    autoCompleteResults: [],
    sendAutoCompleteRequest: null,
    clearAutoComplete: null,
    autoCompleteCategories: [],
    setAllSearchOptions: () => {},
    isDisabled: false,
    prependAutocompleteQuery: [],
};

// This is a legacy search component, that will be removed soon as we move everything to URLSearchInput.
// For now, some of the code is duplicated, and some of the components are referenced from URLSearchInput
// in order to avoid unnecessarily excessive code duplication.

/**
 * Gets the last category search option in the search options
 *
 * @param {!Object[]} searchOptions
 * @returns {!string}
 *
 */
export function getLastCategoryInSearchOptions(searchOptions) {
    const categoryIndex = findLastIndex(searchOptions, ['type', 'categoryOption']);
    if (categoryIndex === -1) {
        return null;
    }
    const category = searchOptions[categoryIndex]?.value.replace(':', '');
    return category;
}

/**
 * Creates the search modifiers based on an array of category strings
 *
 * @param {!string[]} categories an array of category strings
 * @returns {!Object[]}
 *
 * ex: ["Category"] -> [{ value: "Category:", label: "Category:", type: "categoryOption" }]
 */
export function createSearchModifiers(categories) {
    return categories.map((category) => {
        return {
            value: `${category}:`,
            label: `${category}:`,
            type: 'categoryOption',
        };
    });
}

class SearchInput extends Component {
    static propTypes = searchInputPropTypes;

    static defaultProps = searchInputDefaultProps;

    componentWillUnmount() {
        if (!this.props.isGlobal) {
            this.props.setSearchOptions([]);
        }
    }

    sendAutoCompleteRequest = (searchOptions, input) => {
        this.props.setAllSearchOptions(searchOptions);

        // Don't populate autocomplete if the text box is totally empty,
        // since we want people to see just the chips in that case.
        if (!searchOptions.length && !input.length) {
            if (this.props.clearAutoComplete) {
                this.props.clearAutoComplete();
            }
            return;
        }

        if (this.props.sendAutoCompleteRequest) {
            let options = [...searchOptions];
            if (this.props.prependAutocompleteQuery?.length > 0) {
                options = [...this.props.prependAutocompleteQuery, ...searchOptions];
            }
            const clonedSearchOptions = options.slice();
            if (clonedSearchOptions.length === 0) {
                clonedSearchOptions.push(this.props.defaultOption);
            }
            clonedSearchOptions.push({ label: input, value: input });
            const query = searchOptionsToQuery(clonedSearchOptions);
            const queryObj = { query };
            if (this.props.autoCompleteCategories.length) {
                queryObj.categories = this.props.autoCompleteCategories;
            }
            this.props.sendAutoCompleteRequest(queryObj);
        }
    };

    updateAutoCompleteState = (input) => {
        if (!this.queryIsPossiblyBeingTyped()) {
            if (this.props.clearAutoComplete) {
                this.props.clearAutoComplete();
            }
            return;
        }
        this.sendAutoCompleteRequest(this.props.searchOptions, input);
    };

    setOptions = (_, searchOptions) => {
        const actualSearchOptions = removeValuesForKey(this.props.searchOptions, searchOptions);

        // If there is a default option and one search value given, then potentially prepend the default search option
        if (
            this.props.defaultOption &&
            actualSearchOptions.length === 1 &&
            !this.props.searchModifiers.find((x) => x.value === actualSearchOptions[0].value)
        ) {
            actualSearchOptions[0].label = this.trimDefaultOptionFromValueIfExists(
                actualSearchOptions[0].label
            );
            actualSearchOptions.unshift(this.props.defaultOption);
        }
        this.props.setSearchOptions(actualSearchOptions);
        if (this.props.onSearch) {
            this.props.onSearch(actualSearchOptions);
        }
        this.sendAutoCompleteRequest(actualSearchOptions, '');
    };

    queryIsPossiblyBeingTyped = () => {
        // If they're typing into an empty box, then a query is only valid if there's a default option.
        if (this.props.searchOptions.length === 0) {
            return !!this.props.defaultOption;
        }
        return true;
    };

    shouldShowModifiers = () =>
        !this.props.searchOptions.length ||
        this.props.searchOptions[this.props.searchOptions.length - 1].type !== 'categoryOption';

    formatValueWithDefaultOption = (value) => `${this.props.defaultOption.label} ${value}`;

    trimDefaultOptionFromValueIfExists = (value) => {
        const prefix = `${this.props.defaultOption.label} `;
        if (value.startsWith(prefix)) {
            return value.slice(prefix.length);
        }
        return value;
    };

    getSuggestions = () => {
        const { searchOptions, searchModifiers } = this.props;
        let suggestions = [];

        if (this.shouldShowModifiers()) {
            suggestions = suggestions.concat(searchModifiers);
        }

        if (this.queryIsPossiblyBeingTyped()) {
            // If you previously typed a search modifier (Cluster:, Deployment Name:, etc.) then show autocomplete suggestions
            suggestions = suggestions.concat(
                this.props.autoCompleteResults.map((value) => {
                    let modifiedValue = value;
                    if (searchOptions.length === 0) {
                        modifiedValue = this.formatValueWithDefaultOption(modifiedValue);
                    }
                    return { value, label: modifiedValue };
                })
            );
        }

        return suggestions;
    };

    render() {
        const Placeholder = placeholderCreator(this.props.placeholder);
        const { searchOptions, className, isDisabled } = this.props;
        const suggestions = this.getSuggestions();
        const hideDropdown = suggestions.length ? '' : 'hide-dropdown';
        const props = {
            'aria-label': this.props.placeholder,
            isDisabled,
            className: `${className} ${hideDropdown}`,
            components: { ValueContainer, Option, Placeholder, MultiValue },
            options: suggestions,
            optionValue: searchOptions,
            onChange: this.setOptions,
            isMulti: true,
            onInputChange: this.updateAutoCompleteState,
            noOptionsMessage,
            closeMenuOnSelect: false,
            formatCreateLabel: (inputValue) => {
                if (this.props.defaultOption && this.props.searchOptions.length === 0) {
                    return this.formatValueWithDefaultOption(inputValue);
                }
                return inputValue;
            },
            isValidNewOption: (inputValue, _, selectOptions) => {
                if (!inputValue) {
                    return false;
                }
                if (!this.queryIsPossiblyBeingTyped()) {
                    return false;
                }

                // Don't show the new option if it's the same as the top suggestion.
                if (inputMatchesTopOption(inputValue, selectOptions)) {
                    return false;
                }

                // We only allow them to add new options if none of the chips match.
                // Otherwise it might be confusing.
                if (
                    selectOptions.find(
                        (option) =>
                            option.type === 'categoryOption' &&
                            option.label.toLowerCase().startsWith(inputValue.toLowerCase())
                    )
                ) {
                    return false;
                }
                return true;
            },
            createOptionPosition,
        };
        return <Creatable {...props} components={{ ...props.components }} autoFocus />;
    }
}

export default SearchInput;
