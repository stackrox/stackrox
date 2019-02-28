import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { components } from 'react-select';
import { withRouter } from 'react-router-dom';
import queryString from 'qs';

import * as Icon from 'react-feather';
import { Creatable } from 'Components/ReactSelect';
import Query from 'Components/ThrowingQuery';

import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

const borderClass = 'border border-primary-300';
const categoryOptionClass = `bg-primary-200 text-primary-700 ${borderClass}`;
const valueOptionClass = `bg-base-200 text-base-600 ${borderClass}`;

export const placeholderCreator = placeholderText => () => (
    <span className="text-base-500 flex h-full items-center pointer-events-none">
        <span className="font-600 absolute text-lg">{placeholderText}</span>
    </span>
);

const isCategoryChip = value => value.endsWith(':');

export const Option = ({ children, ...rest }) => {
    let className;
    if (isCategoryChip(children)) {
        className = 'bg-primary-200 text-primary-700';
    } else {
        className = 'bg-base-200 text-base-600';
    }
    return (
        <components.Option {...rest}>
            <div className="flex">
                <span
                    className={`${className} border-2 border-primary-300 rounded-sm p-1 px-2 text-sm`}
                >
                    {children}
                </span>
            </div>
        </components.Option>
    );
};

export const ValueContainer = ({ ...props }) => (
    <React.Fragment>
        <span className="text-base-500 flex h-full items-center pl-2 pr-1 pointer-events-none">
            <Icon.Search color="currentColor" size={18} />
        </span>
        <components.ValueContainer {...props} />
    </React.Fragment>
);

export const MultiValue = props => (
    <components.MultiValue
        {...props}
        className={`${
            props.data.type === 'categoryOption' ? categoryOptionClass : valueOptionClass
        } ${props.data.ignore ? 'hidden' : ''}`}
    />
);

export const noOptionsMessage = () => null;

export const createOptionPosition = 'first';

// This function checks whether the input value is the same as that of the top option.
// We don't want to display the user-typed suggestion in this case as it would be a duplicate.
export const inputMatchesTopOption = (input, selectOptions) =>
    selectOptions.length && selectOptions[0].label === input;

class URLSearchInputWithAutocomplete extends Component {
    static propTypes = {
        className: PropTypes.string,
        placeholder: PropTypes.string,
        categoryOptions: PropTypes.arrayOf(PropTypes.string),
        autoCompleteResults: PropTypes.arrayOf(PropTypes.string),
        fetchAutocomplete: PropTypes.func,
        clearAutocomplete: PropTypes.func
    };

    static defaultProps = {
        className: '',
        placeholder: 'Add one or more filters',
        categoryOptions: [],
        autoCompleteResults: [],
        fetchAutocomplete: null,
        clearAutocomplete: null
    };

    createCategoryOption = category => ({
        label: `${category}:`,
        value: `${category}:`,
        type: 'categoryOption',
        ignore: category === 'groupBy'
    });

    getCategory = option => option.value.replace(':', '');

    isCategoryOption = option => option.type === 'categoryOption';

    createValueOption = (value, key) => ({
        label: `${value}`,
        value: `${value}`,
        ignore: key === 'groupBy',
        __isNew__: true
    });

    getValue = option => option.value;

    transformCategoryOptions = options => options.map(this.createCategoryOption);

    transformSearchOptionsToQueryString = searchOptions => {
        const { search: prevSearch } = this.props.location;
        const prevQueryJSON = queryString.parse(prevSearch, { ignoreQueryPrefix: true });
        const queryJSON = {};
        let categoryKey = '';
        searchOptions.forEach(option => {
            if (this.isCategoryOption(option)) {
                const category = this.getCategory(option);
                categoryKey = category;
                queryJSON[categoryKey] = '';
            } else {
                const value = this.getValue(option);
                if (!queryJSON[categoryKey]) {
                    queryJSON[categoryKey] = [value];
                } else {
                    queryJSON[categoryKey].push(value);
                }
            }
        });
        // to not clear the `groupBy` query. will need to remove once search officially supports groupBy
        if (prevQueryJSON.groupBy) queryJSON.groupBy = prevQueryJSON.groupBy;
        const search = queryString.stringify(queryJSON, { arrayFormat: 'repeat' });
        return search;
    };

    transformQueryStringToSearchOptions = search => {
        const queryStringOptions = [];
        const queryJSON = queryString.parse(search, { ignoreQueryPrefix: true });
        Object.keys(queryJSON).forEach(key => {
            const matchedOptionKey = this.props.categoryOptions.find(
                category => category.toLowerCase() === key.toLowerCase()
            );
            if (matchedOptionKey) {
                queryStringOptions.push(this.createCategoryOption(matchedOptionKey));
                const value = queryJSON[key];
                if (Array.isArray(value)) {
                    value.forEach(v => {
                        queryStringOptions.push(this.createValueOption(v));
                    });
                } else if (value && value !== '') {
                    queryStringOptions.push(this.createValueOption(value, key));
                }
            }
        });
        return queryStringOptions;
    };

    replaceLocationSearch = searchOptions => {
        const { pathname } = this.props.location;
        const search = this.transformSearchOptionsToQueryString(searchOptions);
        this.props.history.replace({
            pathname,
            search
        });
    };

    setOptions = (_, searchOptions) => {
        this.replaceLocationSearch(searchOptions);
        this.updateAutocompleteState(searchOptions)('');
    };

    getOptions = () => {
        const { categoryOptions, location } = this.props;
        const searchOptions = this.transformQueryStringToSearchOptions(location.search);
        let options = [];
        if (
            searchOptions.length === 0 ||
            searchOptions[searchOptions.length - 1].type !== 'categoryOption'
        ) {
            options = options.concat(this.transformCategoryOptions(categoryOptions));
        }
        if (searchOptions.length) {
            options = options.concat(this.props.autoCompleteResults.map(this.createValueOption));
        }
        return options;
    };

    updateAutocompleteState = searchOptions => input => {
        if (searchOptions.length === 0) {
            if (this.props.clearAutocomplete) {
                this.props.clearAutocomplete();
            }
            return;
        }
        if (this.props.fetchAutocomplete) {
            const clonedSearchOptions = searchOptions.slice();
            clonedSearchOptions.push(this.createValueOption(input));
            const query = searchOptionsToQuery(clonedSearchOptions);
            this.props.fetchAutocomplete({ query });
        }
    };

    render() {
        const { placeholder, className, location, ...rest } = this.props;
        const Placeholder = placeholderCreator(placeholder);
        const searchOptions = this.transformQueryStringToSearchOptions(location.search);
        const options = this.getOptions();
        const hideDropdown = options.length ? '' : 'hide-dropdown';
        const props = {
            className: `${className} ${hideDropdown}`,
            components: { ValueContainer, Option, Placeholder, MultiValue },
            options,
            optionValue: searchOptions,
            onChange: this.setOptions,
            isMulti: true,
            noOptionsMessage,
            onInputChange: this.updateAutocompleteState(searchOptions),
            defaultMenuIsOpen: searchOptions.length > 0,
            isValidNewOption: (input, _, availableOptions) =>
                input &&
                searchOptions.length > 0 &&
                !inputMatchesTopOption(input, availableOptions),
            formatCreateLabel: inputValue => inputValue,
            createOptionPosition,
            ...rest
        };
        return <Creatable {...props} components={{ ...props.components }} autoFocus />;
    }
}

// eslint-disable-next-line react/no-multi-comp
class URLSearchInput extends Component {
    static propTypes = {
        categories: PropTypes.arrayOf(PropTypes.string)
    };

    static defaultProps = {
        categories: []
    };

    constructor(props) {
        super(props);
        this.state = {
            autoCompleteQuery: ''
        };
    }

    clearAutocomplete = () => {
        this.setState({
            autoCompleteQuery: ''
        });
    };

    fetchAutocomplete = ({ query }) => {
        this.setState({
            autoCompleteQuery: query
        });
    };

    render() {
        if (!this.state.autoCompleteQuery) {
            return (
                <URLSearchInputWithAutocomplete
                    fetchAutocomplete={this.fetchAutocomplete}
                    {...this.props}
                />
            );
        }
        return (
            <Query
                query={SEARCH_AUTOCOMPLETE_QUERY}
                variables={{
                    query: this.state.autoCompleteQuery,
                    categories: this.props.categories
                }}
            >
                {({ data }) => {
                    const autoCompleteResults = data.searchAutocomplete || [];
                    return (
                        <URLSearchInputWithAutocomplete
                            autoCompleteResults={autoCompleteResults}
                            clearAutocomplete={this.clearAutocomplete}
                            fetchAutocomplete={this.fetchAutocomplete}
                            {...this.props}
                        />
                    );
                }}
            </Query>
        );
    }
}

export default withRouter(URLSearchInput);
