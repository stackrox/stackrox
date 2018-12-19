import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import NoResultsMessage from 'Components/NoResultsMessage';

import Table, {
    wrapClassName,
    defaultHeaderClassName,
    defaultColumnClassName
} from 'Components/Table';

import Panel from 'Components/Panel';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import TablePagination from 'Components/TablePagination';

import { selectors } from 'reducers';
import { actions as secretsActions } from 'reducers/secrets';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import SecretDetails, { secretTypeEnumMapping } from './SecretDetails';

class SecretPage extends Component {
    static propTypes = {
        secrets: PropTypes.arrayOf(PropTypes.object).isRequired,
        selectedSecret: PropTypes.shape({
            id: PropTypes.string.isRequired
        }),
        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
        location: ReactRouterPropTypes.location.isRequired,
        isViewFiltered: PropTypes.bool.isRequired
    };

    static defaultProps = {
        selectedSecret: null
    };

    constructor(props) {
        super(props);
        this.state = {
            page: 0
        };
    }

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            this.props.history.push('/main/secrets');
        }
    };

    setTablePage = newPage => {
        this.setState({ page: newPage });
    };

    updateSelectedSecret = secret => {
        const urlSuffix = secret && secret.id ? `/${secret.id}` : '';
        this.props.history.push({
            pathname: `/main/secrets${urlSuffix}`,
            search: this.props.location.search
        });
    };

    renderPanel = () => {
        const { length } = this.props.secrets;
        const paginationComponent = (
            <TablePagination
                page={this.state.page}
                dataLength={length}
                setPage={this.setTablePage}
            />
        );
        const headerText = `${length} Secret${length === 1 ? '' : 's'} ${
            this.props.isViewFiltered ? 'Matched' : ''
        }`;
        return (
            <Panel header={headerText} headerComponents={paginationComponent}>
                <div className="w-full">{this.renderTable()}</div>
            </Panel>
        );
    };

    renderTable = () => {
        const columns = [
            {
                accessor: 'name',
                Header: 'Name',
                headerClassName: `w-1/8 min-w-48 ${defaultHeaderClassName}`,
                className: `w-1/8 min-w-48 word-break-all ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ value }) => <span>{value}</span>
            },
            {
                id: 'createdAt',
                accessor: d => dateFns.format(d.createdAt, dateTimeFormat),
                Header: 'Created',
                headerClassName: `${defaultHeaderClassName}`,
                className: `${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ value }) => <span>{value}</span>
            },
            {
                id: 'types',
                accessor: d => d.types.map(v => secretTypeEnumMapping[v]).join(', '),
                Header: 'Types'
            },
            { accessor: 'clusterName', Header: 'Cluster' },
            { accessor: 'namespace', Header: 'Namespace' }
        ];
        const { secrets, selectedSecret } = this.props;
        const rows = secrets;
        const id = selectedSecret && selectedSecret.id;
        if (!rows.length)
            return <NoResultsMessage message="No results found. Please refine your search." />;
        return (
            <Table
                rows={rows}
                columns={columns}
                onRowClick={this.updateSelectedSecret}
                selectedRowId={id}
                noDataText="No results found. Please refine your search."
                page={this.state.page}
            />
        );
    };

    renderSidePanel = () => {
        const { selectedSecret } = this.props;
        if (!selectedSecret) return null;
        return (
            <Panel
                className="bg-primary-200 z-10 w-full h-full absolute pin-r pin-t md:w-1/2 min-w-72 md:relative"
                header={selectedSecret.name}
                onClose={this.updateSelectedSecret}
            >
                <SecretDetails secret={selectedSecret} />
            </Panel>
        );
    };

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        const defaultOption = this.props.searchModifiers.find(x => x.value === 'Secret:');
        return (
            <section className="flex flex-1 flex-col h-full">
                <div className="flex flex-1 flex-col">
                    <PageHeader header="Secrets" subHeader={subHeader}>
                        <SearchInput
                            className="w-full"
                            id="secrets"
                            searchOptions={this.props.searchOptions}
                            searchModifiers={this.props.searchModifiers}
                            searchSuggestions={this.props.searchSuggestions}
                            setSearchOptions={this.props.setSearchOptions}
                            setSearchModifiers={this.props.setSearchModifiers}
                            setSearchSuggestions={this.props.setSearchSuggestions}
                            onSearch={this.onSearch}
                            defaultOption={defaultOption}
                        />
                    </PageHeader>
                    <div className="flex flex-1 relative">
                        <div className="rounded-sm shadow border-primary-300 bg-base-100 w-full overflow-hidden">
                            {this.renderPanel()}
                        </div>
                        {this.renderSidePanel()}
                    </div>
                </div>
            </section>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getSecretsSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const getSelectedSecret = (state, props) => {
    const { secretId } = props.match.params;
    return secretId ? selectors.getSecret(state, secretId) : null;
};

const mapStateToProps = createStructuredSelector({
    secrets: selectors.getFilteredSecrets,
    selectedSecret: getSelectedSecret,
    searchOptions: selectors.getSecretsSearchOptions,
    searchModifiers: selectors.getSecretsSearchModifiers,
    searchSuggestions: selectors.getSecretsSearchSuggestions,
    isViewFiltered
});

const mapDispatchToProps = {
    setSearchOptions: secretsActions.setSecretsSearchOptions,
    setSearchModifiers: secretsActions.setSecretsSearchModifiers,
    setSearchSuggestions: secretsActions.setSecretsSearchSuggestions
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(SecretPage);
