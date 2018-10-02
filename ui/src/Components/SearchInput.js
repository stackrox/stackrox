import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { components } from 'react-select';
import * as Icon from 'react-feather';

import Select, { Creatable } from 'Components/ReactSelect';

const placeholderCreator = placeholderText => () => (
    <span className="text-base-500 flex flex-1 h-full items-center">
        <Icon.Search color="currentColor" size={18} />
        <span className="font-600 px-2">{placeholderText}</span>
    </span>
);

const Option = ({ children, ...rest }) => (
    <components.Option {...rest}>
        <div className="flex px-3">
            <span className="search-option-categories px-2 text-sm">{children}</span>
        </div>
    </components.Option>
);

const MultiValue = props => (
    <components.MultiValue
        {...props}
        className={
            props.data.type === 'categoryOption'
                ? 'bg-primary-200 border border-primary-300 text-primary-600'
                : 'bg-base-100 border border-base-300 text-base-600'
        }
    />
);

// hack-y (uses internal API of react-select): put cross ahead of the value label
// TODO-ivan: change the behavior to have "key: value" to be one chip
const MultiValueContainer = ({ children, ...rest }) => (
    <components.MultiValueContainer {...rest}>
        {React.Children.toArray(children)[1]}
        {React.Children.toArray(children)[0]}
    </components.MultiValueContainer>
);

const EmptyCreatableMenu = () => null;

class SearchInput extends Component {
    static propTypes = {
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
        placeholder: 'Add one or more resource filters',
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

    setOptions = (_, searchOptions) => {
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

    render() {
        const Placeholder = placeholderCreator(this.props.placeholder);
        const { searchOptions, searchSuggestions } = this.props;

        const props = {
            className: this.props.className,
            components: { Option, Placeholder, MultiValue, MultiValueContainer },
            options: searchSuggestions,
            optionValue: searchOptions,
            onChange: this.setOptions,
            isMulti: true
        };
        if (this.props.searchOptions.length === 0) return <Select {...props} autoFocus />;

        return (
            <Creatable
                {...props}
                components={{ ...props.components, Menu: EmptyCreatableMenu }}
                autoFocus
            />
        );
    }
}

export default SearchInput;
