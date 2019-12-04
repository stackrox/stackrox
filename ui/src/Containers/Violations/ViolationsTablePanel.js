import React from 'react';
import PropTypes from 'prop-types';

import * as Icon from 'react-feather';

import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import { pageSize } from 'Components/TableV2';
import TablePagination from 'Components/TablePaginationV2';
import TableHeader from 'Components/TableHeader';
import ViolationsTable from './ViolationsTable';
import dialogues from './dialogues';

function ViolationsTablePanelTextHeader({ violationsCount, isViewFiltered, checkedAlertIds }) {
    return (
        <TableHeader
            length={violationsCount}
            selectionCount={checkedAlertIds.length}
            type="Violation"
            isViewFiltered={isViewFiltered}
        />
    );
}

ViolationsTablePanelTextHeader.propTypes = {
    violationsCount: PropTypes.number.isRequired,
    isViewFiltered: PropTypes.bool.isRequired,
    checkedAlertIds: PropTypes.arrayOf(PropTypes.string).isRequired
};

function ViolationsTablePanelButtons({ setDialogue, checkedAlertIds, runtimeAlerts }) {
    // Handle dialogue pop ups.
    function showResolveConfirmationDialog() {
        setDialogue(dialogues.resolve);
    }
    function showWhitelistConfirmationDialog() {
        setDialogue(dialogues.whitelist);
    }

    let checkedRuntimeAlerts = 0;
    checkedAlertIds.forEach(id => {
        if (runtimeAlerts.has(id)) checkedRuntimeAlerts += 1;
    });
    const whitelistCount = checkedAlertIds.length;
    return (
        <React.Fragment>
            {checkedRuntimeAlerts !== 0 && (
                <PanelButton
                    icon={<Icon.Check className="h-4 ml-1" />}
                    className="btn btn-base"
                    onClick={showResolveConfirmationDialog}
                    tooltip={`Mark as Resolved (${checkedRuntimeAlerts})`}
                >
                    {`Mark as Resolved (${checkedRuntimeAlerts})`}
                </PanelButton>
            )}
            {whitelistCount !== 0 && (
                <PanelButton
                    icon={<Icon.BellOff className="h-4 ml-1" />}
                    className="btn btn-base"
                    onClick={showWhitelistConfirmationDialog}
                    tooltip={`Whitelist (${whitelistCount})`}
                >
                    {`Whitelist (${whitelistCount})`}
                </PanelButton>
            )}
        </React.Fragment>
    );
}

ViolationsTablePanelButtons.propTypes = {
    setDialogue: PropTypes.func.isRequired,
    checkedAlertIds: PropTypes.arrayOf(PropTypes.string).isRequired,
    runtimeAlerts: PropTypes.shape({
        has: PropTypes.func.isRequired
    }).isRequired
};

function ViolationsTablePanel({
    violations,
    violationsCount,
    isViewFiltered,
    setDialogue,
    selectedAlertId,
    setSelectedAlertId,
    checkedAlertIds,
    setCheckedAlertIds,
    currentPage,
    setCurrentPage,
    setSortOption,
    runtimeAlerts
}) {
    // Currently selected rows in the table.
    const headerTextComponent = (
        <ViolationsTablePanelTextHeader
            violationsCount={violationsCount}
            isViewFiltered={isViewFiltered}
            checkedAlertIds={checkedAlertIds}
        />
    );

    // Handle page changes.
    function changePage(newPage) {
        if (newPage !== currentPage) {
            setCurrentPage(newPage);
        }
    }

    const pageCount = Math.ceil(violationsCount / pageSize);
    const headerComponents = (
        <>
            <ViolationsTablePanelButtons
                setDialogue={setDialogue}
                checkedAlertIds={checkedAlertIds}
                runtimeAlerts={runtimeAlerts}
            />
            <TablePagination pageCount={pageCount} page={currentPage} setPage={changePage} />
        </>
    );

    return (
        <Panel headerTextComponent={headerTextComponent} headerComponents={headerComponents}>
            <div className="h-full w-full">
                <ViolationsTable
                    violations={violations}
                    selectedAlertId={selectedAlertId}
                    setSelectedAlertId={setSelectedAlertId}
                    selectedRows={checkedAlertIds}
                    setSelectedRows={setCheckedAlertIds}
                    setSortOption={setSortOption}
                />
            </div>
        </Panel>
    );
}

ViolationsTablePanel.propTypes = {
    violations: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    violationsCount: PropTypes.number.isRequired,
    isViewFiltered: PropTypes.bool.isRequired,
    setDialogue: PropTypes.func.isRequired,
    selectedAlertId: PropTypes.string,
    setSelectedAlertId: PropTypes.func.isRequired,
    checkedAlertIds: PropTypes.arrayOf(PropTypes.string).isRequired,
    setCheckedAlertIds: PropTypes.func.isRequired,
    currentPage: PropTypes.number.isRequired,
    setCurrentPage: PropTypes.func.isRequired,
    setSortOption: PropTypes.func.isRequired,
    runtimeAlerts: PropTypes.shape({
        has: PropTypes.func.isRequired
    }).isRequired
};

ViolationsTablePanel.defaultProps = {
    selectedAlertId: undefined
};

export default ViolationsTablePanel;
