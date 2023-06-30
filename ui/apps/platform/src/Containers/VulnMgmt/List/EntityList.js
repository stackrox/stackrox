import React, { useContext, useRef, useLayoutEffect } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import resolvePath from 'object-resolve-path';
import workflowStateContext from 'Containers/workflowStateContext';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import URLSearchInput from 'Components/URLSearchInput';
import { withRouter } from 'react-router-dom';
import { searchCategories } from 'constants/entityTypes';
import createPDFTable from 'utils/pdfUtils';
import CheckboxTable from 'Components/CheckboxTable';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';

import {
    entityCountNounOrdinaryCase,
    entityNounOrdinaryCasePlural,
} from '../entitiesForVulnerabilityManagement';

const EntityList = ({
    autoFocusSearchInput,
    entityType,
    history,
    idAttribute,
    rowData,
    searchOptions,
    sort,
    selectedRowId,
    tableColumns,
    SubComponent,
    defaultExpanded,
    checkbox,
    selection,
    setSelection,
    tableHeaderComponents,
    renderRowActionButtons,
    serverSidePagination,
    onSortedChange,
    disableSortRemove,
    page,
    totalResults,
    pageSize,
}) => {
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

    function setPage(newPage) {
        if (typeof setSelection === 'function') {
            setSelection([]);
        }

        history.push(workflowState.setPage(newPage).toUrl());
    }

    // render section
    const noDataText = `No ${entityNounOrdinaryCasePlural[entityType]} found. Please refine your search.`;

    const header = entityCountNounOrdinaryCase(totalResults, entityType);

    const placeholder = `Filter ${entityNounOrdinaryCasePlural[entityType]}`;

    // need `useLayoutEffect` here to solve an edge case,
    //   where the use have navigated to a single-page sublist,
    //   then clicked "Open in current window",
    //   then clicked the browser's back button
    //   see: https://stack-rox.atlassian.net/browse/ROX-4450
    useLayoutEffect(() => {
        if (rowData.length) {
            const query = {}; // TODO: improve sep. of concerns in pdfUtils
            createPDFTable(rowData, entityType, query, 'capture-list', tableColumns);
        }
    }, [entityType, rowData, tableColumns]);

    const availableCategories = [searchCategories[entityType]];
    const headerComponents = (
        <>
            <div className="flex flex-1 justify-start">
                <URLSearchInput
                    placeholder={placeholder}
                    className="w-full"
                    categoryOptions={searchOptions}
                    categories={availableCategories}
                    autoFocus={autoFocusSearchInput}
                />
            </div>
            <div className="ml-2 flex">{tableHeaderComponents}</div>
            <TablePagination
                page={page}
                dataLength={totalResults}
                pageSize={pageSize}
                setPage={setPage}
            />
        </>
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
            SubComponent={SubComponent}
            expanded={defaultExpanded}
            manual={serverSidePagination}
            sorted={sort}
            onSortedChange={onSortedChange}
            disableSortRemove={disableSortRemove}
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
                SubComponent={SubComponent}
                expanded={defaultExpanded}
                selection={selection}
                ref={tableRef}
                toggleRow={toggleTableRow}
                toggleSelectAll={toggleAllTableRows}
                renderRowActionButtons={renderRowActionButtons}
                manual={serverSidePagination}
                sorted={sort}
                onSortedChange={onSortedChange}
                disableSortRemove={disableSortRemove}
            />
        );
    }

    return (
        <PanelNew testid="panel">
            <PanelHead>
                <PanelTitle testid="panel-header" text={header} />
                <PanelHeadEnd>{headerComponents}</PanelHeadEnd>
            </PanelHead>
            <PanelBody>
                <div className="bg-base-100 h-full w-full">{tableComponent}</div>
            </PanelBody>
        </PanelNew>
    );
};

EntityList.propTypes = {
    autoFocusSearchInput: PropTypes.bool,
    entityType: PropTypes.string.isRequired,
    idAttribute: PropTypes.string.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    rowData: PropTypes.arrayOf(PropTypes.shape({})),
    searchOptions: PropTypes.arrayOf(PropTypes.string),
    sort: PropTypes.arrayOf(PropTypes.shape({})),
    selectedRowId: PropTypes.string,
    tableColumns: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    SubComponent: PropTypes.func,
    defaultExpanded: PropTypes.arrayOf(PropTypes.shape({})),
    checkbox: PropTypes.bool,
    selection: PropTypes.arrayOf(PropTypes.string),
    setSelection: PropTypes.func,
    tableHeaderComponents: PropTypes.element,
    renderRowActionButtons: PropTypes.func,
    serverSidePagination: PropTypes.bool,
    onSortedChange: PropTypes.func,
    disableSortRemove: PropTypes.bool,
    page: PropTypes.number,
    totalResults: PropTypes.number,
    pageSize: PropTypes.number,
};

EntityList.defaultProps = {
    autoFocusSearchInput: true,
    rowData: null,
    searchOptions: [],
    sort: null,
    selectedRowId: null,
    SubComponent: null,
    defaultExpanded: null,
    checkbox: false,
    selection: [],
    setSelection: null,
    tableHeaderComponents: null,
    renderRowActionButtons: null,
    serverSidePagination: false,
    onSortedChange: null,
    disableSortRemove: false,
    page: 0,
    totalResults: 0,
    pageSize: null,
};

export default withRouter(EntityList);
