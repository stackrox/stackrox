import React, { useState } from 'react';
import { withRouter } from 'react-router-dom';
import pluralize from 'pluralize';
import resolvePath from 'object-resolve-path';

import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import Panel from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import entityLabels from 'messages/entity';
import URLService from 'modules/URLService';
import createPDFTable from 'utils/pdfUtils';
import isGQLLoading from 'utils/gqlLoading';

const VulnMgmtTable = ({
    wrapperClass,
    headerText,
    query,
    entityType,
    tableColumns,
    createTableRows,
    selectedRowId,
    idAttribute,
    defaultSorted,
    loading,
    error,
    data,
    match,
    location,
    history
}) => {
    const [page, setPage] = useState(0);

    function onRowClickHandler(row) {
        const id = resolvePath(row, idAttribute);
        const url = URLService.getURL(match, location)
            .push(id)
            .url();
        history.push(url);
    }

    function getRenderComponents(headerComponents, tableRows) {
        const header = `${tableRows.length} ${pluralize(
            headerText || entityLabels[entityType],
            tableRows.length
        )}`;

        const noDataText = `No ${pluralize(
            entityLabels[entityType]
        )} found. Please refine your search.`;

        return (
            <Panel className={wrapperClass} header={header} headerComponents={headerComponents}>
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
            </Panel>
        );
    }

    function getHeaderComponents(totalRows) {
        return (
            <>
                <div className="flex flex-1 justify-start">
                    <span>URLSearchInput goes here</span>
                </div>
                {/* TODO: update pagination to use server-side pagination */}
                <TablePagination page={page} dataLength={totalRows} setPage={setPage} />
            </>
        );
    }

    if (isGQLLoading(loading, data)) return <Loader />;

    if (error || !data) return <PageNotFound resourceType={entityType} />;

    const tableRows = createTableRows(data);

    // TODO: fix big StackRox logo on PDF
    if (tableRows.length) {
        createPDFTable(tableRows, entityType, query, 'capture-list', tableColumns);
    }
    const headerComponents = getHeaderComponents(tableRows.length);

    const content = getRenderComponents(headerComponents, tableRows);

    return (
        <section className="h-full w-full bg-base-100" id="capture-list">
            {content}
        </section>
    );
};

export default withRouter(VulnMgmtTable);
