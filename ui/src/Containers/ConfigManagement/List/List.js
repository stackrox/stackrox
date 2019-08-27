import React, { useState } from 'react';
import PropTypes from 'prop-types';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import createPDFTable from 'utils/pdfUtils';
import resolvePath from 'object-resolve-path';

import NoResultsMessage from 'Components/NoResultsMessage';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import Panel from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import URLSearchInput from 'Components/URLSearchInput';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import { searchCategories as searchCategoryTypes } from 'constants/entityTypes';
import URLService from 'modules/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';

const List = ({
    className,
    headerText,
    query,
    variables,
    entityType,
    tableColumns,
    createTableRows,
    selectedRowId,
    idAttribute,
    defaultSorted,
    defaultSearchOptions,
    data,
    match,
    location,
    history
}) => {
    const [page, setPage] = useState(0);
    const message = `No ${pluralize(entityType.toLowerCase())} were found for this entity`;

    function onRowClickHandler(row) {
        const id = resolvePath(row, idAttribute);
        const url = URLService.getURL(match, location)
            .push(id)
            .url();
        history.push(url);
    }

    const categories = [searchCategoryTypes[entityType]];

    function getRenderComponents(headerComponents, tableRows) {
        const header = `${tableRows.length} ${pluralize(
            headerText || entityLabels[entityType],
            tableRows.length
        )}`;

        return (
            <section id="capture-list" className="h-full w-full bg-base-100">
                <Panel className={className} header={header} headerComponents={headerComponents}>
                    <Table
                        rows={tableRows}
                        columns={tableColumns}
                        onRowClick={onRowClickHandler}
                        idAttribute={idAttribute}
                        id="capture-list"
                        selectedRowId={selectedRowId}
                        noDataText="No results found. Please refine your search."
                        page={page}
                        defaultSorted={defaultSorted}
                    />
                </Panel>
            </section>
        );
    }

    function getHeaderComponents(totalRows) {
        return (
            <>
                <div className="flex flex-1 justify-start">
                    <Query query={SEARCH_OPTIONS_QUERY} action="list" variables={{ categories }}>
                        {({ data: results }) => {
                            const searchOptions =
                                results && results.searchOptions
                                    ? [...results.searchOptions, ...defaultSearchOptions]
                                    : [];
                            return (
                                <URLSearchInput
                                    className="w-full"
                                    categoryOptions={searchOptions}
                                    categories={categories}
                                />
                            );
                        }}
                    </Query>
                </div>
                <TablePagination page={page} dataLength={totalRows} setPage={setPage} />
            </>
        );
    }

    if (data) {
        if (data.length === 0 && !variables) return <NoResultsMessage message={message} />;
        const headerComponents = getHeaderComponents(data.length);
        createPDFTable(data, entityType, query, 'capture-list', tableColumns);
        return getRenderComponents(headerComponents, data);
    }

    return (
        <Query query={query} variables={variables}>
            {({ loading, data: queryData }) => {
                if (loading) return <Loader />;
                if (!queryData) return <PageNotFound resourceType={entityType} />;
                const tableRows = createTableRows(queryData) || [];

                if (tableRows.length === 0 && !variables)
                    return <NoResultsMessage message={message} />;
                const headerComponents = getHeaderComponents(tableRows.length);

                if (tableRows.length) {
                    createPDFTable(tableRows, entityType, query, 'capture-list', tableColumns);
                }
                return getRenderComponents(headerComponents, tableRows);
            }}
        </Query>
    );
};

List.propTypes = {
    className: PropTypes.string,
    query: PropTypes.shape().isRequired,
    variables: PropTypes.shape(),
    entityType: PropTypes.string.isRequired,
    tableColumns: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    createTableRows: PropTypes.func.isRequired,
    selectedRowId: PropTypes.string,
    idAttribute: PropTypes.string.isRequired,
    headerText: PropTypes.string,
    defaultSorted: PropTypes.arrayOf(PropTypes.shape({})),
    defaultSearchOptions: PropTypes.arrayOf(PropTypes.string),
    data: PropTypes.arrayOf(PropTypes.shape({})),
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired
};

List.defaultProps = {
    className: '',
    variables: {},
    headerText: '',
    selectedRowId: null,
    defaultSorted: [],
    defaultSearchOptions: [],
    data: null
};

export default withRouter(List);
