import React, { Component } from 'react';
import PropTypes from 'prop-types';

import { Creatable } from 'react-select';
import * as Icon from 'react-feather';
import differenceBy from 'lodash/differenceBy';

class SearchInput extends Component {
    static propTypes = {
        className: PropTypes.string,
        placeholder: PropTypes.string,
        searchOptions: PropTypes.arrayOf(PropTypes.object),
        searchModifiers: PropTypes.arrayOf(PropTypes.object),
        searchSuggestions: PropTypes.arrayOf(PropTypes.object),
        setSearchOptions: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired
    };

    static defaultProps = {
        placeholder: 'Page filters',
        className: '',
        searchOptions: [],
        searchModifiers: [],
        searchSuggestions: []
    };

    onInputChange = value => value;

    setOptions = searchOptions => {
        console.log(searchOptions);
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

    promptTextCreator = label => `Search for: "${label}"`;

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
        return (
            <Creatable
                className={`search-input ${this.props.className}`}
                name="search-input"
                placeholder={searchIcon}
                onInputChange={this.onInputChange}
                options={searchSuggestions}
                optionRenderer={this.renderOption}
                arrowRenderer={this.renderArrow}
                value={searchOptions}
                onChange={this.setOptions}
                promptTextCreator={this.promptTextCreator}
                filterOptions={this.filterOptions}
                ignoreCase
                multi
            />
        );
    }
}

export default SearchInput;
