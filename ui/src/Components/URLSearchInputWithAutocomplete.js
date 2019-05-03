import React from 'react';
import PropTypes from 'prop-types';
import { components } from 'react-select';
import queryString from 'qs';
import { connect } from 'react-redux';
import { actions as searchAutoCompleteActions } from 'reducers/searchAutocomplete';

import * as Icon from 'react-feather';
import { Creatable } from 'Components/ReactSelect';
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
            <Icon.Filter color="currentColor" size={18} />
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

const URLSearchInputWithAutocomplete = props => {
    function createCategoryOption(category) {
        return {
            label: `${category}:`,
            value: `${category}:`,
            type: 'categoryOption',
            ignore: category === 'groupBy'
        };
    }

    function getCategory(option) {
        return option.value.replace(':', '');
    }

    function isCategoryOption(option) {
        return option.type === 'categoryOption';
    }

    function createValueOption(value, key) {
        return {
            label: `${value}`,
            value: `${value}`,
            ignore: key === 'groupBy',
            __isNew__: true
        };
    }

    function getValue(option) {
        return option.value;
    }

    function transformCategoryOptions(options) {
        return options.map(createCategoryOption);
    }

    function transformSearchOptionsToQueryString(searchOptions) {
        const { search: prevSearch } = props.location;
        const prevQueryJSON = queryString.parse(prevSearch, { ignoreQueryPrefix: true });
        const queryJSON = {};
        let categoryKey = '';
        searchOptions.forEach(option => {
            if (isCategoryOption(option)) {
                const category = getCategory(option);
                categoryKey = category;
                queryJSON[categoryKey] = '';
            } else {
                const value = getValue(option);
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
    }

    function transformQueryStringToSearchOptions(search) {
        const queryStringOptions = [];
        const queryJSON = queryString.parse(search, { ignoreQueryPrefix: true });
        Object.keys(queryJSON).forEach(key => {
            const matchedOptionKey = props.categoryOptions.find(
                category => category.toLowerCase() === key.toLowerCase()
            );
            if (matchedOptionKey) {
                queryStringOptions.push(createCategoryOption(matchedOptionKey));
                const value = queryJSON[key];
                if (Array.isArray(value)) {
                    value.forEach(v => {
                        queryStringOptions.push(createValueOption(v));
                    });
                } else if (value && value !== '') {
                    queryStringOptions.push(createValueOption(value, key));
                }
            }
        });
        return queryStringOptions;
    }

    function replaceLocationSearch(searchOptions) {
        const { pathname } = props.location;
        const search = transformSearchOptionsToQueryString(searchOptions);
        props.history.replace({
            pathname,
            search
        });
    }

    function updateAutocompleteState(searchOptions) {
        return input => {
            props.setAllSearchOptions(searchOptions);
            if (searchOptions.length === 0) {
                if (props.clearAutocomplete) {
                    props.clearAutocomplete();
                }
                return;
            }
            if (props.fetchAutocomplete) {
                const clonedSearchOptions = searchOptions.slice();
                clonedSearchOptions.push(createValueOption(input));
                const query = searchOptionsToQuery(clonedSearchOptions);
                props.fetchAutocomplete({ query });
            }
        };
    }

    function setOptions(_, searchOptions) {
        replaceLocationSearch(searchOptions);
        updateAutocompleteState(searchOptions)('');
    }

    function getOptions() {
        const { categoryOptions, location } = props;
        const searchOptions = transformQueryStringToSearchOptions(location.search);
        let options = [];
        if (
            searchOptions.length === 0 ||
            searchOptions[searchOptions.length - 1].type !== 'categoryOption'
        ) {
            options = options.concat(transformCategoryOptions(categoryOptions));
        }
        if (searchOptions.length) {
            options = options.concat(props.autoCompleteResults.map(createValueOption));
        }
        return options;
    }

    const { placeholder, className, location, ...rest } = props;
    const Placeholder = placeholderCreator(placeholder);
    const searchOptions = transformQueryStringToSearchOptions(location.search);
    const options = getOptions();
    const hideDropdown = options.length ? '' : 'hide-dropdown';
    const creatableProps = {
        className: `${className} ${hideDropdown}`,
        components: { ValueContainer, Option, Placeholder, MultiValue },
        options,
        optionValue: searchOptions,
        onChange: setOptions,
        isMulti: true,
        noOptionsMessage,
        onInputChange: updateAutocompleteState(searchOptions),
        defaultMenuIsOpen: searchOptions.length > 0,
        isValidNewOption: (input, _, availableOptions) =>
            input && searchOptions.length > 0 && !inputMatchesTopOption(input, availableOptions),
        formatCreateLabel: inputValue => inputValue,
        createOptionPosition,
        ...rest
    };
    return (
        <Creatable {...creatableProps} components={{ ...creatableProps.components }} autoFocus />
    );
};

URLSearchInputWithAutocomplete.propTypes = {
    className: PropTypes.string,
    placeholder: PropTypes.string,
    categoryOptions: PropTypes.arrayOf(PropTypes.string),
    autoCompleteResults: PropTypes.arrayOf(PropTypes.string),
    fetchAutocomplete: PropTypes.func,
    clearAutocomplete: PropTypes.func,
    setAllSearchOptions: PropTypes.func.isRequired
};

URLSearchInputWithAutocomplete.defaultProps = {
    className: '',
    placeholder: 'Add one or more filters',
    categoryOptions: [],
    autoCompleteResults: [],
    fetchAutocomplete: null,
    clearAutocomplete: null
};

const mapDispatchToProps = {
    setAllSearchOptions: searchAutoCompleteActions.setAllSearchOptions
};

export default connect(
    null,
    mapDispatchToProps
)(URLSearchInputWithAutocomplete);
