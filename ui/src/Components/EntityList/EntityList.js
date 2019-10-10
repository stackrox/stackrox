import React, { useState } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import pluralize from 'pluralize';
import resolvePath from 'object-resolve-path';

import Panel from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import URLSearchInput from 'Components/URLSearchInput';

import { searchCategories } from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import URLService from 'modules/URLService';
import createPDFTable from 'utils/pdfUtils';

const EntityList = ({
    autoFocusSearchInput,
    defaultSorted,
    entityType,
    headerText,
    history,
    idAttribute,
    location,
    match,
    rowData,
    searchOptions,
    selectedRowId,
    tableColumns,
    wrapperClass
}) => {
    const [page, setPage] = useState(0);

    function onRowClickHandler(row) {
        const id = resolvePath(row, idAttribute);
        const url = URLService.getURL(match, location)
            .push(id)
            .url();
        history.push(url);
    }

    function getHeaderComponents(totalRows, categoryOptions, categories, autoFocus) {
        return (
            <>
                <div className="flex flex-1 justify-start">
                    <URLSearchInput
                        className="w-full"
                        categoryOptions={categoryOptions}
                        categories={categories}
                        autoFocus={autoFocus}
                    />
                </div>
                {/* TODO: update pagination to use server-side pagination */}
                <TablePagination page={page} dataLength={totalRows} setPage={setPage} />
            </>
        );
    }

    // TODO: fix big StackRox logo on PDF
    if (rowData.length) {
        const query = {}; // TODO: improve sep. of concerns in pdfUtils
        createPDFTable(rowData, entityType, query, 'capture-list', tableColumns);
    }

    // render section
    const entityLabel = entityLabels[entityType] || 'results';
    const noDataText = `No ${pluralize(entityLabel)} found. Please refine your search.`;

    const header = `${rowData.length} ${pluralize(
        headerText || entityLabels[entityType],
        rowData.length
    )}`;

    const availableCategories = [searchCategories[entityType]];
    const headerComponents = getHeaderComponents(
        rowData.length,
        searchOptions,
        availableCategories,
        autoFocusSearchInput
    );

    return (
        <Panel className={wrapperClass} header={header} headerComponents={headerComponents}>
            <Table
                rows={rowData}
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
};

EntityList.propTypes = {
    autoFocusSearchInput: PropTypes.bool,
    defaultSorted: PropTypes.arrayOf(PropTypes.shape({})),
    entityType: PropTypes.string.isRequired,
    headerText: PropTypes.string,
    idAttribute: PropTypes.string.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
    rowData: PropTypes.arrayOf(PropTypes.shape({})),
    searchOptions: PropTypes.arrayOf(PropTypes.string),
    selectedRowId: PropTypes.string,
    tableColumns: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    wrapperClass: PropTypes.string
};

EntityList.defaultProps = {
    autoFocusSearchInput: true,
    defaultSorted: [],
    headerText: '',
    rowData: null,
    searchOptions: [],
    selectedRowId: null,
    wrapperClass: ''
};

export default withRouter(EntityList);
