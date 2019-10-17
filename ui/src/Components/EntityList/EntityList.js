import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import pluralize from 'pluralize';
import resolvePath from 'object-resolve-path';
import WorkflowStateMgr from 'modules/WorkflowStateManager';
import workflowStateContext from 'Containers/workflowStateContext';
import { generateURL } from 'modules/URLReadWrite';
import Panel from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import URLSearchInput from 'Components/URLSearchInput';
import { withRouter } from 'react-router-dom';
import { searchCategories } from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import createPDFTable from 'utils/pdfUtils';

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
    tableColumns
}) => {
    const [page, setPage] = useState(0);
    const workflowState = useContext(workflowStateContext);

    function onRowClickHandler(row) {
        const id = resolvePath(row, idAttribute);
        const workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.pushListItem(id);
        const url = generateURL(workflowStateMgr.workflowState);

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
        <Panel
            className={selectedRowId ? 'bg-base-100 overlay' : ''}
            header={header}
            headerComponents={headerComponents}
        >
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
    rowData: PropTypes.arrayOf(PropTypes.shape({})),
    searchOptions: PropTypes.arrayOf(PropTypes.string),
    selectedRowId: PropTypes.string,
    tableColumns: PropTypes.arrayOf(PropTypes.shape({})).isRequired
};

EntityList.defaultProps = {
    autoFocusSearchInput: true,
    defaultSorted: [],
    headerText: '',
    rowData: null,
    searchOptions: [],
    selectedRowId: null
};

export default withRouter(EntityList);
