import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as deploymentsActions, types } from 'reducers/deployments';

import NoResultsMessage from 'Components/NoResultsMessage';
import ReactRowSelectTable from 'Components/ReactRowSelectTable';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';
import { sortNumber } from 'sorters/sorters';
import RiskDetails from './RiskDetails';
import DeploymentDetails from './DeploymentDetails';

class RiskPage extends Component {
    static propTypes = {
        deployments: PropTypes.arrayOf(PropTypes.object).isRequired,
        selectedDeployment: PropTypes.shape({
            id: PropTypes.string.isRequired
        }),
        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        isViewFiltered: PropTypes.bool.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
        location: ReactRouterPropTypes.location.isRequired,
        isFetchingDeployment: PropTypes.bool
    };

    static defaultProps = {
        isFetchingDeployment: false,
        selectedDeployment: null
    };

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            this.props.history.push('/main/risk');
        }
    };

    updateSelectedDeployment = deployment => {
        const urlSuffix = deployment && deployment.id ? `/${deployment.id}` : '';
        this.props.history.push({
            pathname: `/main/risk${urlSuffix}`,
            search: this.props.location.search
        });
    };

    renderTable() {
        const columns = [
            {
                Header: 'Name',
                accessor: 'name'
            },
            {
                Header: 'Cluster',
                accessor: 'cluster'
            },
            {
                Header: 'Namespace',
                accessor: 'namespace'
            },
            {
                Header: 'Priority',
                accessor: 'priority',
                sortMethod: sortNumber('priority')
            }
        ];

        const { deployments, selectedDeployment } = this.props;
        const rows = deployments;
        const id = selectedDeployment && selectedDeployment.id;
        if (!rows.length)
            return <NoResultsMessage message="No results found. Please refine your search." />;
        return (
            <ReactRowSelectTable
                rows={rows}
                columns={columns}
                onRowClick={this.updateSelectedDeployment}
                selectedRowId={id}
                noDataText="No results found. Please refine your search."
            />
        );
    }

    renderSidePanel = () => {
        const { selectedDeployment } = this.props;
        if (!selectedDeployment) return null;

        const riskPanelTabs = [{ text: 'Risk Indicators' }, { text: 'Deployment Details' }];
        const content = this.props.isFetchingDeployment ? (
            <Loader />
        ) : (
            <Tabs headers={riskPanelTabs}>
                <TabContent>
                    <div className="flex flex-1 flex-col">
                        <RiskDetails risk={selectedDeployment.risk} />
                    </div>
                </TabContent>
                <TabContent>
                    <div className="flex flex-1 flex-col relative">
                        <div className="absolute w-full">
                            <DeploymentDetails deployment={selectedDeployment} />
                        </div>
                    </div>
                </TabContent>
            </Tabs>
        );

        return (
            <div className="w-2/3">
                <Panel header={selectedDeployment.name} onClose={this.updateSelectedDeployment}>
                    {content}
                </Panel>
            </div>
        );
    };

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 flex-col">
                    <PageHeader header="Risk" subHeader={subHeader}>
                        <SearchInput
                            className="flex flex-1"
                            id="risk"
                            searchOptions={this.props.searchOptions}
                            searchModifiers={this.props.searchModifiers}
                            searchSuggestions={this.props.searchSuggestions}
                            setSearchOptions={this.props.setSearchOptions}
                            setSearchModifiers={this.props.setSearchModifiers}
                            setSearchSuggestions={this.props.setSearchSuggestions}
                            onSearch={this.onSearch}
                        />
                    </PageHeader>
                    <div className="flex flex-1">
                        <div className="w-full pl-3 pt-3 pr-3 bg-white rounded-sm shadow bg-base-100">
                            {this.renderTable()}
                        </div>
                        {this.renderSidePanel()}
                    </div>
                </div>
            </section>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getDeploymentsSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const getSelectedDeployment = (state, props) => {
    const { deploymentId } = props.match.params;
    return deploymentId ? selectors.getSelectedDeployment(state, deploymentId) : null;
};

const mapStateToProps = createStructuredSelector({
    deployments: selectors.getFilteredDeployments,
    selectedDeployment: getSelectedDeployment,
    searchOptions: selectors.getDeploymentsSearchOptions,
    searchModifiers: selectors.getDeploymentsSearchModifiers,
    searchSuggestions: selectors.getDeploymentsSearchSuggestions,
    isViewFiltered,
    isFetchingDeployment: state => selectors.getLoadingStatus(state, types.FETCH_DEPLOYMENT)
});

const mapDispatchToProps = {
    setSearchOptions: deploymentsActions.setDeploymentsSearchOptions,
    setSearchModifiers: deploymentsActions.setDeploymentsSearchModifiers,
    setSearchSuggestions: deploymentsActions.setDeploymentsSearchSuggestions
};

export default connect(mapStateToProps, mapDispatchToProps)(RiskPage);
