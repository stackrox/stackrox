import React, { Component } from 'react';
import PropTypes from 'prop-types';

import Select from 'react-select';
import * as Icon from 'react-feather';
import differenceBy from 'lodash/differenceBy';

class SearchInput extends Component {
    static propTypes = {
        id: PropTypes.string,
        className: PropTypes.string,
        placeholder: PropTypes.string,
        searchOptions: PropTypes.arrayOf(PropTypes.object),
        searchModifiers: PropTypes.arrayOf(PropTypes.object),
        searchSuggestions: PropTypes.arrayOf(PropTypes.object),
        setSearchOptions: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        onSearch: PropTypes.func,
        isGlobal: PropTypes.bool
    };

    static defaultProps = {
        id: '',
        placeholder: 'Page filters',
        className: '',
        searchOptions: [],
        searchModifiers: [],
        searchSuggestions: [],
        onSearch: null,
        isGlobal: false
    };

    componentWillUnmount() {
        if (!this.props.isGlobal) this.props.setSearchOptions([]);
    }

    onInputChange = value => value;

    setOptions = searchOptions => {
        const searchModifiers = this.props.searchModifiers.slice();
        let searchSuggestions = [];
        if (searchOptions.length && searchOptions[searchOptions.length - 1].type) {
            // If you previously typed a search modifier (Cluster:, Deployment Name:, etc.) then don't show any search suggestions
            searchSuggestions = [];
        } else {
            searchSuggestions = searchModifiers;
        }
        this.props.setSearchOptions(searchOptions);
        this.props.setSearchSuggestions(searchSuggestions);
        if (this.props.onSearch) this.props.onSearch(searchOptions);
    };

    filterOptions = (options, filterString, excludeOptions, props) => {
        let filterValue = filterString.slice();
        if (props.ignoreCase) filterValue = filterValue.toLowerCase();
        try {
            return differenceBy(
                options.filter(obj => {
                    let { label } = obj;
                    if (props.ignoreCase) label = label.toLowerCase();
                    return label.match(filterValue) && !obj.className;
                }), // Don't show any newly created options
                excludeOptions.filter(obj => obj.type),
                'value'
            );
        } catch (error) {
            return [];
        }
    };

    renderArrow = () => (
        <span className="text-base-400">
            <Icon.ChevronDown color="currentColor" size={18} />
        </span>
    );

    renderOption = option => (
        <div className="flex px-3">
            <span className="search-option-categories px-2 text-sm">{option.label}</span>
        </div>
    );

    render() {
        const searchIcon = (
            <span className="text-base-400 flex flex-1 h-full items-center">
                <Icon.Search color="currentColor" size={18} />
                <span className="font-600 px-1">{this.props.placeholder}</span>
            </span>
        );
        const searchOptions = this.props.searchOptions.slice();
        const searchSuggestions = this.props.searchSuggestions.slice();
        const props = {
            className: this.props.id
                ? `${this.props.id}-search-input ${this.props.className}`
                : `search-input ${this.props.className}`,
            name: 'search-input',
            placeholder: searchIcon,
            onInputChange: this.onInputChange,
            options: searchSuggestions,
            optionRenderer: this.renderOption,
            arrowRenderer: this.renderArrow,
            value: searchOptions,
            onChange: this.setOptions,
            filterOptions: this.filterOptions,
            ignoreCase: true,
            multi: true
        };
        if (this.props.searchOptions.length === 0) return <Select {...props} autoFocus />;
        return <Select.Creatable {...props} autoFocus />;
    }
}

export default SearchInput;
