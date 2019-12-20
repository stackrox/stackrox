import React, { useState, useContext, useRef } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import pluralize from 'pluralize';
import resolvePath from 'object-resolve-path';
import workflowStateContext from 'Containers/workflowStateContext';
import Panel from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import URLSearchInput from 'Components/URLSearchInput';
import { withRouter } from 'react-router-dom';
import { searchCategories } from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import createPDFTable from 'utils/pdfUtils';
import CheckboxTable from 'Components/CheckboxTable';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';

const EntityList = ({
    autoFocusSearchInput,
    defaultSorted,
    entityType,
    headerText,
    history,
    idAttribute,
    rowData,
    searchOptions,
    selectedRowId,
    tableColumns,
    SubComponent,
    defaultExpanded,
    checkbox,
    selection,
    setSelection,
    tableHeaderComponents,
    renderRowActionButtons
}) => {
    const [page, setPage] = useState(0);
    const tableRef = useRef(null);
    const workflowState = useContext(workflowStateContext);

    function toggleTableRow(id) {
        const newSelection = toggleRow(id, selection);
        setSelection(newSelection);
    }

    function toggleAllTableRows() {
        const rowsLength = selection.length;
        const ref = tableRef.current.reactTable;
        const newSelection = toggleSelectAll(rowsLength, selection, ref);
        setSelection(newSelection);
    }

    function onRowClickHandler(row) {
        const id = resolvePath(row, idAttribute);
        const url = workflowState.pushListItem(id).toUrl();

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
                <div className="ml-2 flex">{tableHeaderComponents}</div>
                {/* TODO: update pagination to use server-side pagination */}
                <TablePagination page={page} dataLength={totalRows} setPage={setPage} />
            </>
        );
    }

    // render section
    const entityLabel = entityLabels[entityType] || 'results';
    const noDataText = `No ${pluralize(entityLabel)} found. Please refine your search.`;

    const header = `${rowData.length} ${pluralize(
        headerText || entityLabels[entityType],
        rowData.length
    )}`;

    // TODO: fix big StackRox logo on PDF
    if (rowData.length) {
        const query = {}; // TODO: improve sep. of concerns in pdfUtils
        createPDFTable(rowData, entityType, query, 'capture-list', tableColumns);
    }

    const availableCategories = [searchCategories[entityType]];
    const headerComponents = getHeaderComponents(
        rowData.length,
        searchOptions,
        availableCategories,
        autoFocusSearchInput
    );
    let tableComponent = (
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
            SubComponent={SubComponent}
            expanded={defaultExpanded}
        />
    );

    if (checkbox) {
        tableComponent = (
            <CheckboxTable
                rows={rowData}
                columns={tableColumns}
                onRowClick={onRowClickHandler}
                idAttribute={idAttribute}
                id="capture-list"
                selectedRowId={selectedRowId}
                noDataText={noDataText}
                page={page}
                defaultSorted={defaultSorted}
                SubComponent={SubComponent}
                expanded={defaultExpanded}
                selection={selection}
                ref={tableRef}
                toggleRow={toggleTableRow}
                toggleSelectAll={toggleAllTableRows}
                renderRowActionButtons={renderRowActionButtons}
            />
        );
    }

    return (
        <Panel
            className={selectedRowId ? 'bg-base-100 overlay' : ''}
            header={header}
            headerComponents={headerComponents}
        >
            {tableComponent}
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
    rowData: PropTypes.arrayOf(PropTypes.shape({})),
    searchOptions: PropTypes.arrayOf(PropTypes.string),
    selectedRowId: PropTypes.string,
    tableColumns: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    SubComponent: PropTypes.func,
    defaultExpanded: PropTypes.arrayOf(PropTypes.shape({})),
    checkbox: PropTypes.bool,
    selection: PropTypes.arrayOf(PropTypes.string),
    setSelection: PropTypes.func,
    tableHeaderComponents: PropTypes.element,
    renderRowActionButtons: PropTypes.func
};

EntityList.defaultProps = {
    autoFocusSearchInput: true,
    defaultSorted: [],
    headerText: '',
    rowData: null,
    searchOptions: [],
    selectedRowId: null,
    SubComponent: null,
    defaultExpanded: null,
    checkbox: false,
    selection: [],
    setSelection: null,
    tableHeaderComponents: null,
    renderRowActionButtons: null
};

export default withRouter(EntityList);
