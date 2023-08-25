import React, { useState } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import pluralize from 'pluralize';
import resolvePath from 'object-resolve-path';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import Table, { DEFAULT_PAGE_SIZE } from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import URLSearchInput from 'Components/URLSearchInput';
import { searchCategories as searchCategoryTypes } from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import entityLabels from 'messages/entity';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import isGQLLoading from 'utils/gqlLoading';
import createPDFTable from 'utils/pdfUtils';
import URLService from 'utils/URLService';

const ListFrontendPaginated = ({
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
    autoFocusSearchInput,
    noDataText,
    match,
    location,
    history,
}) => {
    const [page, setPage] = useState(0);

    function onRowClickHandler(row) {
        const id = resolvePath(row, idAttribute);
        const url = URLService.getURL(match, location).push(id).url();
        history.push(url);
    }

    const categories = [searchCategoryTypes[entityType]];
    const placeholder = `Filter ${pluralize(entityLabels[entityType])}`;

    function getRenderComponents(headerComponents, tableRows) {
        const header = `${tableRows.length} ${pluralize(
            headerText || entityLabels[entityType],
            tableRows.length
        )}`;

        return (
            <PanelNew testid="panel">
                <PanelHead>
                    <PanelTitle testid="panel-header" text={header} />
                    <PanelHeadEnd>{headerComponents}</PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <Table
                        rows={tableRows}
                        columns={tableColumns}
                        onRowClick={onRowClickHandler}
                        idAttribute={idAttribute}
                        id="capture-list"
                        selectedRowId={selectedRowId}
                        noDataText={noDataText}
                        page={page}
                        defaultSorted={defaultSorted}
                    />
                </PanelBody>
            </PanelNew>
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
                                    placeholder={placeholder}
                                    className="w-full"
                                    categoryOptions={searchOptions}
                                    categories={categories}
                                    autoFocus={autoFocusSearchInput}
                                />
                            );
                        }}
                    </Query>
                </div>
                <TablePagination
                    page={page}
                    dataLength={totalRows}
                    setPage={setPage}
                    pageSize={DEFAULT_PAGE_SIZE}
                />
            </>
        );
    }

    if (data) {
        const headerComponents = getHeaderComponents(data.length);
        if (data.length) {
            createPDFTable(data, entityType, query, 'capture-list', tableColumns);
        }
        return getRenderComponents(headerComponents, data);
    }

    return (
        <section className="h-full w-full" id="capture-list">
            <Query query={query} variables={variables}>
                {({ loading, data: queryData }) => {
                    if (isGQLLoading(loading, data)) {
                        return <Loader />;
                    }
                    if (!queryData) {
                        return (
                            <PageNotFound
                                resourceType={entityType}
                                useCase={useCases.CONFIG_MANAGEMENT}
                            />
                        );
                    }
                    const tableRows = createTableRows(queryData) || [];
                    const headerComponents = getHeaderComponents(tableRows.length);

                    if (tableRows.length) {
                        createPDFTable(tableRows, entityType, query, 'capture-list', tableColumns);
                    }
                    return getRenderComponents(headerComponents, tableRows);
                }}
            </Query>
        </section>
    );
};

ListFrontendPaginated.propTypes = {
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
    autoFocusSearchInput: PropTypes.bool,
    noDataText: PropTypes.string,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
};

ListFrontendPaginated.defaultProps = {
    variables: {},
    headerText: '',
    selectedRowId: null,
    defaultSorted: [],
    defaultSearchOptions: [],
    data: null,
    autoFocusSearchInput: true,
    noDataText: 'No results found. Please refine your search.',
};

export default withRouter(ListFrontendPaginated);
