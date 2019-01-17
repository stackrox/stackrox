import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { components } from 'react-select';
import { withRouter } from 'react-router-dom';
import queryString from 'qs';

import * as Icon from 'react-feather';
import { Creatable } from 'Components/ReactSelect';

const borderClass = 'border border-primary-300';
const categoryOptionClass = `bg-primary-200 text-primary-700 ${borderClass}`;
const valueOptionClass = `bg-base-200 text-base-600 ${borderClass}`;

const placeholderCreator = placeholderText => () => (
    <span className="text-base-500 flex h-full items-center pointer-events-none">
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
        className={props.data.type === 'categoryOption' ? categoryOptionClass : valueOptionClass}
    />
);

const noOptionsMessage = () => null;

class URLSearchInput extends Component {
    static propTypes = {
        className: PropTypes.string,
        placeholder: PropTypes.string,
        categoryOptions: PropTypes.arrayOf(PropTypes.string)
    };

    static defaultProps = {
        className: '',
        placeholder: 'Add one or more filters',
        categoryOptions: []
    };

    createCategoryOption = category => ({
        label: `${category}:`,
        value: `${category}:`,
        type: 'categoryOption'
    });

    getCategory = option => option.value.replace(':', '');

    isCategoryOption = option => option.type === 'categoryOption';

    createValueOption = value => ({
        label: `${value}`,
        value: `${value}`,
        __isNew__: true
    });

    getValue = option => option.value;

    transformCategoryOptions = options => options.map(option => this.createCategoryOption(option));

    transformSearchOptionsToQueryString = searchOptions => {
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
        const search = queryString.stringify(queryJSON, { encode: false, arrayFormat: 'repeat' });
        return search;
    };

    transformQueryStringToSearchOptions = search => {
        const queryStringOptions = [];
        const queryJSON = queryString.parse(search, { ignoreQueryPrefix: true });
        Object.keys(queryJSON).forEach(key => {
            if (this.props.categoryOptions.find(category => category === key)) {
                queryStringOptions.push(this.createCategoryOption(key));
                const value = queryJSON[key];
                if (Array.isArray(value)) {
                    value.forEach(v => {
                        queryStringOptions.push(this.createValueOption(v));
                    });
                } else if (value && value !== '') {
                    queryStringOptions.push(this.createValueOption(value));
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
        if (
            searchOptions.length === 1 &&
            !this.transformCategoryOptions(this.props.categoryOptions).find(
                x => x.value === searchOptions[0].value
            )
        ) {
            searchOptions.unshift();
        }
        this.replaceLocationSearch(searchOptions);
    };

    getOptions = () => {
        const { categoryOptions, location } = this.props;
        const searchOptions = this.transformQueryStringToSearchOptions(location.search);
        let options = [];
        if (searchOptions.length && searchOptions[searchOptions.length - 1].type) {
            // If you previously typed a search modifier (Cluster:, Deployment Name:, etc.) then don't show any search suggestions
            options = [];
        } else {
            options = this.transformCategoryOptions(categoryOptions);
        }
        return options;
    };

    render() {
        const { placeholder, className, location, ...rest } = this.props;
        const Placeholder = placeholderCreator(placeholder);
        const searchOptions = this.transformQueryStringToSearchOptions(location.search);
        const hideDropdown = this.getOptions().length ? '' : 'hide-dropdown';
        const props = {
            className: `${className} ${hideDropdown}`,
            components: { ValueContainer, Option, Placeholder, MultiValue },
            options: this.getOptions(),
            optionValue: searchOptions,
            onChange: this.setOptions,
            isMulti: true,
            noOptionsMessage,
            isValidNewOption: inputValue => {
                if (searchOptions.length === 0) {
                    return false;
                }
                return inputValue;
            },
            ...rest
        };
        return <Creatable {...props} components={{ ...props.components }} autoFocus />;
    }
}

export default withRouter(URLSearchInput);
