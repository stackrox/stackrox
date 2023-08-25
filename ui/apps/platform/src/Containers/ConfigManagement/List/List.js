import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import pluralize from 'pluralize';
import resolvePath from 'object-resolve-path';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import URLSearchInput from 'Components/URLSearchInput';
import configMgmtPaginationContext from 'Containers/configMgmtPaginationContext';
import workflowStateContext from 'Containers/workflowStateContext';
import { searchCategories as searchCategoryTypes } from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import entityLabels from 'messages/entity';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import isGQLLoading from 'utils/gqlLoading';
import createPDFTable from 'utils/pdfUtils';
import queryService from 'utils/queryService';
import URLService from 'utils/URLService';

const serverSidePagination = true;

const List = ({
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
    totalResults,
    autoFocusSearchInput,
    noDataText,
    match,
    location,
    history,
}) => {
    const workflowState = useContext(workflowStateContext);
    const configMgmtPagination = useContext(configMgmtPaginationContext);
    const page = workflowState.paging[configMgmtPagination.pageParam];
    const pageSort = workflowState.sort[configMgmtPagination.sortParam];
    const tableSort = pageSort || defaultSorted;

    const [sortFields, setSortFields] = useState({});

    function onRowClickHandler(row) {
        const id = resolvePath(row, idAttribute);
        const url = URLService.getURL(match, location).push(id).url();
        history.push(url);
    }

    const categories = [searchCategoryTypes[entityType]];
    const placeholder = `Filter ${pluralize(entityLabels[entityType])}`;

    function getRenderComponents(headerComponents, tableRows, totalCount) {
        const header = `${totalCount} ${pluralize(
            headerText || entityLabels[entityType],
            totalCount
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
                        sorted={tableSort}
                        onSortedChange={onSortedChange}
                        manual={serverSidePagination}
                        disableSortRemove
                    />
                </PanelBody>
            </PanelNew>
        );
    }

    function setPage(newPage) {
        history.push(workflowState.setPage(newPage).toUrl());
    }

    function onSortedChange(newSort, column) {
        const workflowSort = newSort.map((sortItem) => {
            const id = sortFields[sortItem.id] || column.sortField;
            setSortFields({ [sortItem.id]: id, ...sortFields });
            const { desc } = sortItem;
            return {
                id,
                desc,
            };
        });

        const url = workflowState.setSort(workflowSort).toUrl();
        history.push(url);
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
                    pageSize={LIST_PAGE_SIZE}
                />
            </>
        );
    }

    if (data) {
        const headerComponents = getHeaderComponents(totalResults);
        if (data.length) {
            createPDFTable(data, entityType, query, 'capture-list', tableColumns);
        }
        return getRenderComponents(headerComponents, data, totalResults);
    }

    const varsWithPagination = {
        ...variables,
        pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE),
    };
    return (
        <section className="h-full w-full" id="capture-list">
            <Query query={query} variables={varsWithPagination}>
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
                    const totalCount = queryData?.count || 0;
                    const headerComponents = getHeaderComponents(totalCount);

                    if (tableRows.length) {
                        createPDFTable(tableRows, entityType, query, 'capture-list', tableColumns);
                    }
                    return getRenderComponents(headerComponents, tableRows, totalCount);
                }}
            </Query>
        </section>
    );
};

List.propTypes = {
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
    totalResults: PropTypes.number,
    autoFocusSearchInput: PropTypes.bool,
    noDataText: PropTypes.string,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
};

List.defaultProps = {
    variables: {},
    headerText: '',
    selectedRowId: null,
    defaultSorted: [],
    defaultSearchOptions: [],
    data: null,
    totalResults: null,
    autoFocusSearchInput: true,
    noDataText: 'No results found. Please refine your search.',
};

export default withRouter(List);
