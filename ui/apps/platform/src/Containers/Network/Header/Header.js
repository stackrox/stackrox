import React from 'react';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import PageHeader from 'Components/PageHeader';
import NetworkSearch from './NetworkSearch';
import ClusterSelect from './ClusterSelect';
import SimulatorButton from './SimulatorButton';
import TimeWindowSelector from './TimeWindowSelector';
import CIDRFormButton from './CIDRFormButton';

function Header({ isViewFiltered }) {
    const subHeader = isViewFiltered ? 'Filtered view' : 'Default view';
    return (
        <>
            <PageHeader header="Network Graph" subHeader={subHeader} classes="flex-1 border-none">
                <ClusterSelect />
                <NetworkSearch />
                <TimeWindowSelector />
                <SimulatorButton />
            </PageHeader>
            <CIDRFormButton />
        </>
    );
}

const isViewFiltered = createSelector(
    [selectors.getNetworkSearchOptions],
    (searchOptions) => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    isViewFiltered,
});

export default connect(mapStateToProps, null)(Header);
