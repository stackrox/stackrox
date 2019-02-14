import React, { Component } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { components } from 'react-select';
import * as Icon from 'react-feather';
import { createStructuredSelector } from 'reselect';

import { Creatable } from 'Components/ReactSelect';

import { actions as searchAutoCompleteActions } from 'reducers/searchAutocomplete';
import { selectors } from 'reducers';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

const placeholderCreator = placeholderText => () => (
    <span className="text-base-500 flex h-full items-center pointer-events-none text-lg">
        <span className="font-600 absolute">{placeholderText}</span>
    </span>
);

const Option = ({ children, ...rest }) => (
    <components.Option {...rest}>
        <div className="flex">
            <span className="search-option-categories px-2 text-sm">{children}</span>
        </div>
    </components.Option>
);

const ValueContainer = ({ ...props }) => (
    <React.Fragment>
        <span className="text-base-500 flex h-full items-center pl-2 pr-1 pointer-events-none">
            <Icon.Search color="currentColor" size={18} />
        </span>
        <components.ValueContainer {...props} />
    </React.Fragment>
);

const MultiValue = props => (
    <components.MultiValue
        {...props}
        className={
            props.data.type === 'categoryOption'
                ? 'bg-primary-200 border border-primary-300 text-primary-700'
                : 'bg-base-200 border border-base-300 text-base-600'
        }
    />
);

const noOptionsMessage = () => null;

class SearchInput extends Component {
    static propTypes = {
        className: PropTypes.string,
        placeholder: PropTypes.string,
        searchOptions: PropTypes.arrayOf(PropTypes.object),
        searchModifiers: PropTypes.arrayOf(PropTypes.object),
        setSearchOptions: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        onSearch: PropTypes.func,
        isGlobal: PropTypes.bool,
        defaultOption: PropTypes.shape({
            value: PropTypes.string,
            label: PropTypes.string,
            category: PropTypes.string
        }),
        autoCompleteResults: PropTypes.arrayOf(PropTypes.string),
        sendAutoCompleteRequest: PropTypes.func,
        clearAutoComplete: PropTypes.func,
        autoCompleteCategories: PropTypes.arrayOf(PropTypes.string)
    };

    static defaultProps = {
        placeholder: 'Add one or more resource filters',
        className: '',
        searchOptions: [],
        searchModifiers: [],
        onSearch: null,
        isGlobal: false,
        defaultOption: null,
        autoCompleteResults: [],
        sendAutoCompleteRequest: null,
        clearAutoComplete: null,
        autoCompleteCategories: []
    };

    componentWillUnmount() {
        if (!this.props.isGlobal) this.props.setSearchOptions([]);
    }

    sendAutoCompleteRequest = (searchOptions, input) => {
        // Don't populate autocomplete if the text box is totally empty,
        // since we want people to see just the chips in that case.
        if (!searchOptions.length && !input.length) {
            if (this.props.clearAutoComplete) {
                this.props.clearAutoComplete();
            }
            return;
        }

        if (this.props.sendAutoCompleteRequest) {
            const clonedSearchOptions = searchOptions.slice();
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

    updateAutoCompleteState = input => {
        if (!this.queryIsPossiblyBeingTyped()) {
            if (this.props.clearAutoComplete) {
                this.props.clearAutoComplete();
            }
            return;
        }
        this.sendAutoCompleteRequest(this.props.searchOptions, input);
    };

    setOptions = (_, searchOptions) => {
        // If there is a default option and one search value given, then potentially prepend the default search option
        const actualSearchOptions = searchOptions;
        if (
            this.props.defaultOption &&
            actualSearchOptions.length === 1 &&
            !this.props.searchModifiers.find(x => x.value === actualSearchOptions[0].value)
        ) {
            actualSearchOptions[0].label = this.trimDefaultOptionFromValueIfExists(
                actualSearchOptions[0].label
            );
            actualSearchOptions.unshift(this.props.defaultOption);
        }
        this.props.setSearchOptions(actualSearchOptions);
        if (this.props.onSearch) this.props.onSearch(actualSearchOptions);
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

    formatValueWithDefaultOption = value => `${this.props.defaultOption.label} ${value}`;

    trimDefaultOptionFromValueIfExists = value => {
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
                this.props.autoCompleteResults.map(value => {
                    let modifiedValue = value;
                    if (searchOptions.length === 0)
                        modifiedValue = this.formatValueWithDefaultOption(modifiedValue);
                    return { value, label: modifiedValue };
                })
            );
        }

        return suggestions;
    };

    render() {
        const Placeholder = placeholderCreator(this.props.placeholder);
        const { searchOptions, className } = this.props;
        const suggestions = this.getSuggestions();
        const hideDropdown = suggestions.length ? '' : 'hide-dropdown';
        const props = {
            className: `${className} ${hideDropdown}`,
            components: { ValueContainer, Option, Placeholder, MultiValue },
            options: suggestions,
            optionValue: searchOptions,
            onChange: this.setOptions,
            isMulti: true,
            onInputChange: this.updateAutoCompleteState,
            noOptionsMessage,
            closeMenuOnSelect: false,
            formatCreateLabel: inputValue => {
                if (this.props.defaultOption && this.props.searchOptions.length === 0) {
                    return this.formatValueWithDefaultOption(inputValue);
                }
                return inputValue;
            },
            isValidNewOption: (inputValue, _, selectOptions) => {
                if (!inputValue) return false;
                if (!this.queryIsPossiblyBeingTyped()) return false;

                // We only allow them to add new options if none of the chips match.
                // Otherwise it might be confusing.
                if (
                    selectOptions.find(option =>
                        option.label.toLowerCase().startsWith(inputValue.toLowerCase())
                    )
                )
                    return false;
                return true;
            }
        };
        return <Creatable {...props} components={{ ...props.components }} autoFocus />;
    }
}

const mapStateToProps = createStructuredSelector({
    autoCompleteResults: selectors.getAutoCompleteResults
});

const mapDispatchToProps = {
    sendAutoCompleteRequest: searchAutoCompleteActions.sendAutoCompleteRequest,
    clearAutoComplete: searchAutoCompleteActions.clearAutoComplete
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(SearchInput);
