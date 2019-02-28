import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as searchActions } from 'reducers/network/search';

import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import ClusterSelect from './ClusterSelect';
import SimulatorButton from './SimulatorButton';

class Header extends Component {
    static propTypes = {
        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        isViewFiltered: PropTypes.bool.isRequired,
        closeWizard: PropTypes.func.isRequired
    };

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            this.props.closeWizard();
        }
    };

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <PageHeader
                header="Network Graph"
                subHeader={subHeader}
                className="w-2/3 bg-primary-200 "
            >
                <SearchInput
                    id="network"
                    className="w-full"
                    searchOptions={this.props.searchOptions}
                    searchModifiers={this.props.searchModifiers}
                    searchSuggestions={this.props.searchSuggestions}
                    setSearchOptions={this.props.setSearchOptions}
                    setSearchModifiers={this.props.setSearchModifiers}
                    setSearchSuggestions={this.props.setSearchSuggestions}
                    onSearch={this.onSearch}
                />
                <ClusterSelect />
                <SimulatorButton />
            </PageHeader>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getNetworkSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    searchOptions: selectors.getNetworkSearchOptions,
    searchModifiers: selectors.getNetworkSearchModifiers,
    searchSuggestions: selectors.getNetworkSearchSuggestions,
    isViewFiltered
});

const mapDispatchToProps = {
    setSearchOptions: searchActions.setNetworkSearchOptions,
    setSearchModifiers: searchActions.setNetworkSearchModifiers,
    setSearchSuggestions: searchActions.setNetworkSearchSuggestions,
    closeWizard: pageActions.closeNetworkWizard
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Header);
