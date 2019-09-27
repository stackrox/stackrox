import React from 'react';
import { withRouter } from 'react-router-dom';
import pluralize from 'pluralize';
import resolvePath from 'object-resolve-path';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import Panel from 'Components/Panel';
import Table from 'Components/Table';
import entityLabels from 'messages/entity';
import URLService from 'modules/URLService';
import createPDFTable from 'utils/pdfUtils';
import isGQLLoading from 'utils/gqlLoading';

const VulnMgmtTable = ({
    wrapperClass,
    headerText,
    query,
    variables,
    entityType,
    tableColumns,
    createTableRows,
    selectedRowId,
    idAttribute,
    defaultSorted,
    data,
    match,
    location,
    history
}) => {
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
                    defaultSorted={defaultSorted}
                />
            </Panel>
        );
    }

    function getHeaderComponents() {
        return (
            <>
                <div className="flex flex-1 justify-start">
                    <span>URLSearchInput goes here</span>
                </div>
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
        <section className="h-full w-full bg-base-100" id="capture-list">
            <Query query={query} variables={variables}>
                {({ loading, data: queryData }) => {
                    if (isGQLLoading(loading, data)) return <Loader />;
                    if (!queryData) return <PageNotFound resourceType={entityType} />;
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

export default withRouter(VulnMgmtTable);
