import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { components } from 'react-select';
import queryString from 'qs';
import { connect } from 'react-redux';
import * as Icon from 'react-feather';

import { actions as searchAutoCompleteActions } from 'reducers/searchAutocomplete';
import { Creatable } from 'Components/ReactSelect';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import searchContext from 'Containers/searchContext';
import workflowStateContext from 'Containers/workflowStateContext';
import { newWorkflowCases } from 'constants/useCaseTypes';

const borderClass = 'border border-primary-300';
const categoryOptionClass = `bg-primary-200 text-primary-700 ${borderClass}`;
const valueOptionClass = `bg-base-200 text-base-600 ${borderClass}`;

// Render readonly input with placeholder instead of span to prevent insufficient color contrast.
export const placeholderCreator = (placeholderText) =>
    function Placeholder() {
        return (
            <span className="flex h-full items-center pointer-events-none">
                <input
                    className="bg-base-100 text-base-600 absolute pf-u-w-100"
                    placeholder={placeholderText}
                    readOnly
                />
            </span>
        );
    };

const isCategoryChip = (value) => value.endsWith(':');

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
    <>
        <span className="text-base-500 flex h-full items-center pl-2 pr-1 pointer-events-none">
            <Icon.Filter color="currentColor" size={18} />
        </span>
        <components.ValueContainer {...props} />
    </>
);

export const MultiValue = (props) => (
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

export const removeValuesForKey = (oldOptions, newOptions) => {
    const actualSearchOptions = [...newOptions];
    if (oldOptions.length > actualSearchOptions.length) {
        const removedKeyIndex = oldOptions.findIndex(
            (x) =>
                !actualSearchOptions.some((y) => x.value === y.value) && x.type === 'categoryOption'
        );

        if (removedKeyIndex !== -1) {
            let nextKeyIndex = actualSearchOptions.findIndex(
                (x, i) => i >= removedKeyIndex && x.type === 'categoryOption'
            );
            if (removedKeyIndex !== nextKeyIndex) {
                if (nextKeyIndex === -1) {
                    nextKeyIndex = actualSearchOptions.length;
                }
                actualSearchOptions.splice(removedKeyIndex, nextKeyIndex - removedKeyIndex);
            }
        }
    }
    return actualSearchOptions;
};

const URLSearchInputWithAutocomplete = ({
    location,
    history,
    autoCompleteResults,
    categoryOptions,
    setAllSearchOptions,
    clearAutocomplete,
    fetchAutocomplete,
    placeholder,
    className,
    prependAutocompleteQuery,
    ...rest
}) => {
    const searchParam = useContext(searchContext);
    const workflowState = useContext(workflowStateContext);

    function createCategoryOption(category) {
        return {
            label: `${category}:`,
            value: `${category}:`,
            type: 'categoryOption',
            ignore: category === 'groupBy',
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
            __isNew__: true,
        };
    }

    function getValue(option) {
        return option.value;
    }

    function transformCategoryOptions(options) {
        return options.map(createCategoryOption);
    }

    function getFullQueryObject() {
        return queryString.parse(location.search, { ignoreQueryPrefix: true });
    }

    function transformSearchOptionsToQueryString(searchOptions) {
        const currentFullQueryObject = getFullQueryObject();
        const newSearch = {};
        let categoryKey = '';
        searchOptions.forEach((option) => {
            if (isCategoryOption(option)) {
                const category = getCategory(option);
                categoryKey = category;
                newSearch[categoryKey] = '';
            } else {
                const value = getValue(option);
                if (!newSearch[categoryKey]) {
                    newSearch[categoryKey] = value;
                } else {
                    newSearch[categoryKey] = [newSearch[categoryKey]];
                    newSearch[categoryKey].push(value);
                }
            }
        });
        // to not clear the `groupBy` query. will need to remove once search officially supports groupBy
        // if (prevQueryJSON.groupBy) queryJSON.groupBy = prevQueryJSON.groupBy;

        if (newWorkflowCases.includes(workflowState?.useCase)) {
            // Get the full querystring to redirect to
            //   first, check it we have all complete key/value pairs in the search object
            const isCompleteSearch = Object.keys(newSearch).every((k) => !!newSearch[k]);

            //   now, if all are complete, reset to first page, too; otherwise don't
            const url = isCompleteSearch
                ? workflowState.setPage(0).setSearch(newSearch).toUrl()
                : workflowState.setSearch(newSearch).toUrl();
            const qsStart = url.indexOf('?');
            if (qsStart === -1) {
                return '';
            }
            return url.substr(qsStart);
        }

        // For backwards compatibility
        const newQueryObject = { ...currentFullQueryObject, [searchParam]: newSearch };
        return queryString.stringify(newQueryObject, {
            arrayFormat: 'repeat',
            encodeValuesOnly: true,
        });
    }

    function transformQueryStringToSearchOptions() {
        const queryStringOptions = [];
        const fullQueryObject = getFullQueryObject();
        const queryObj = newWorkflowCases.includes(workflowState?.useCase)
            ? workflowState.getCurrentSearchState()
            : fullQueryObject[searchParam] || {};
        Object.keys(queryObj).forEach((key) => {
            const matchedOptionKey = categoryOptions.find(
                (category) => category.toLowerCase() === key.toLowerCase()
            );
            if (matchedOptionKey) {
                queryStringOptions.push(createCategoryOption(matchedOptionKey));
                const value = queryObj[key];
                if (Array.isArray(value)) {
                    value.forEach((v) => {
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
        const { pathname } = location;
        const search = transformSearchOptionsToQueryString(searchOptions);
        history.replace({
            pathname,
            search,
        });
    }

    function updateAutocompleteState(searchOptions) {
        return (input) => {
            setAllSearchOptions(searchOptions);
            if (searchOptions.length === 0) {
                if (clearAutocomplete) {
                    clearAutocomplete();
                }
                return;
            }
            if (fetchAutocomplete) {
                let clonedSearchOptions = [...searchOptions];
                if (prependAutocompleteQuery) {
                    clonedSearchOptions = [...prependAutocompleteQuery, ...searchOptions];
                }
                clonedSearchOptions.push(createValueOption(input));
                const query = searchOptionsToQuery(clonedSearchOptions);
                fetchAutocomplete({ query });
            }
        };
    }

    function setOptions(_, searchOptions) {
        const oldOptions = transformQueryStringToSearchOptions(location.search);
        const actualSearchOptions = removeValuesForKey(oldOptions, searchOptions);

        replaceLocationSearch(actualSearchOptions);
        updateAutocompleteState(actualSearchOptions)('');
    }

    function getOptions() {
        const searchOptions = transformQueryStringToSearchOptions(location.search);
        let options = [];
        if (
            searchOptions.length === 0 ||
            searchOptions[searchOptions.length - 1].type !== 'categoryOption'
        ) {
            options = options.concat(transformCategoryOptions(categoryOptions));
        }
        if (searchOptions.length) {
            options = options.concat(autoCompleteResults.map(createValueOption));
        }
        return options;
    }

    const Placeholder = placeholderCreator(placeholder);
    const searchOptions = transformQueryStringToSearchOptions(location.search);
    const options = getOptions();
    const hideDropdown = options.length ? '' : 'hide-dropdown';
    const isFocused = document.activeElement.id === 'url-search-input';
    const creatableProps = {
        'aria-label': placeholder,
        className: `${className} ${hideDropdown}`,
        components: { ValueContainer, Option, Placeholder, MultiValue },
        options,
        optionValue: searchOptions,
        onChange: setOptions,
        inputId: 'url-search-input',
        isMulti: true,
        noOptionsMessage,
        closeMenuOnSelect: false,
        onInputChange: updateAutocompleteState(searchOptions),
        defaultMenuIsOpen: searchOptions.length > 0 && isFocused,
        isValidNewOption: (input, _, availableOptions) =>
            input && searchOptions.length > 0 && !inputMatchesTopOption(input, availableOptions),
        formatCreateLabel: (inputValue) => inputValue,
        createOptionPosition,
        ...rest,
    };
    return <Creatable {...creatableProps} components={{ ...creatableProps.components }} />;
};

URLSearchInputWithAutocomplete.propTypes = {
    className: PropTypes.string,
    placeholder: PropTypes.string.isRequired,
    categoryOptions: PropTypes.arrayOf(PropTypes.string),
    autoCompleteResults: PropTypes.arrayOf(PropTypes.string),
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    fetchAutocomplete: PropTypes.func,
    clearAutocomplete: PropTypes.func,
    setAllSearchOptions: PropTypes.func.isRequired,
    prependAutocompleteQuery: PropTypes.arrayOf(
        PropTypes.shape({
            value: PropTypes.string,
        })
    ),
};

URLSearchInputWithAutocomplete.defaultProps = {
    className: '',
    categoryOptions: [],
    autoCompleteResults: [],
    fetchAutocomplete: null,
    clearAutocomplete: null,
    prependAutocompleteQuery: [],
};

const mapDispatchToProps = {
    setAllSearchOptions: searchAutoCompleteActions.setAllSearchOptions,
};

export default connect(null, mapDispatchToProps)(URLSearchInputWithAutocomplete);
