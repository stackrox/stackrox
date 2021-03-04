import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import { Tag, Check, BellOff } from 'react-feather';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd } from 'Components/Panel';
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
    checkedAlertIds: PropTypes.arrayOf(PropTypes.string).isRequired,
};

function ViolationsTablePanelButtons({
    setDialogue,
    checkedAlertIds,
    resolvableAlerts,
    excludableAlertIds,
}) {
    // Handle dialogue pop ups.
    function showResolveConfirmationDialog() {
        setDialogue(dialogues.resolve);
    }
    function showExcludeConfirmationDialog() {
        setDialogue(dialogues.excludeScopes);
    }
    function showTagConfirmationDialog() {
        setDialogue(dialogues.tag);
    }

    let checkedResolvableAlerts = 0;
    checkedAlertIds.forEach((id) => {
        if (resolvableAlerts.has(id)) {
            checkedResolvableAlerts += 1;
        }
    });
    const numCheckedAlertIds = checkedAlertIds.length;
    let scopesToExcludeCount = 0;
    checkedAlertIds.forEach((id) => {
        if (excludableAlertIds.has(id)) {
            scopesToExcludeCount += 1;
        }
    });

    return (
        <>
            {numCheckedAlertIds > 0 && (
                <PanelButton
                    icon={<Tag className="h-4 ml-1" />}
                    dataTestId="bulk-add-tags-button"
                    className="btn btn-base ml-2"
                    onClick={showTagConfirmationDialog}
                    tooltip={`Add Tags for ${pluralize(
                        'Violation',
                        numCheckedAlertIds
                    )} (${numCheckedAlertIds})`}
                >
                    {`Add Tags for ${pluralize(
                        'Violation',
                        numCheckedAlertIds
                    )} (${numCheckedAlertIds})`}
                </PanelButton>
            )}
            {checkedResolvableAlerts > 0 && (
                <PanelButton
                    icon={<Check className="h-4 ml-1" />}
                    className="btn btn-base ml-2"
                    onClick={showResolveConfirmationDialog}
                    tooltip={`Mark as Resolved (${checkedResolvableAlerts})`}
                >
                    {`Mark as Resolved (${checkedResolvableAlerts})`}
                </PanelButton>
            )}
            {scopesToExcludeCount > 0 && (
                <PanelButton
                    icon={<BellOff className="h-4 ml-1" />}
                    className="btn btn-base ml-2"
                    onClick={showExcludeConfirmationDialog}
                    tooltip={`Exclude (${scopesToExcludeCount})`}
                >
                    {`Exclude (${scopesToExcludeCount})`}
                </PanelButton>
            )}
        </>
    );
}

ViolationsTablePanelButtons.propTypes = {
    setDialogue: PropTypes.func.isRequired,
    checkedAlertIds: PropTypes.arrayOf(PropTypes.string).isRequired,
    excludableAlertIds: PropTypes.arrayOf(PropTypes.string).isRequired,
    resolvableAlerts: PropTypes.shape({
        has: PropTypes.func.isRequired,
    }).isRequired,
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
    resolvableAlerts,
    excludableAlertIds,
}) {
    // Handle page changes.
    function changePage(newPage) {
        if (newPage !== currentPage) {
            setCurrentPage(newPage);
        }
    }

    return (
        <PanelNew testid="panel">
            <PanelHead>
                <ViolationsTablePanelTextHeader
                    violationsCount={violationsCount}
                    isViewFiltered={isViewFiltered}
                    checkedAlertIds={checkedAlertIds}
                />
                <PanelHeadEnd>
                    <ViolationsTablePanelButtons
                        setDialogue={setDialogue}
                        checkedAlertIds={checkedAlertIds}
                        resolvableAlerts={resolvableAlerts}
                        excludableAlertIds={excludableAlertIds}
                    />
                    <TablePagination
                        pageCount={Math.ceil(violationsCount / pageSize)}
                        page={currentPage}
                        setPage={changePage}
                    />
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>
                <ViolationsTable
                    violations={violations}
                    selectedAlertId={selectedAlertId}
                    setSelectedAlertId={setSelectedAlertId}
                    selectedRows={checkedAlertIds}
                    setSelectedRows={setCheckedAlertIds}
                    setSortOption={setSortOption}
                />
            </PanelBody>
        </PanelNew>
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
    resolvableAlerts: PropTypes.shape({
        has: PropTypes.func.isRequired,
    }).isRequired,
    excludableAlertIds: PropTypes.shape({
        has: PropTypes.func.isRequired,
    }).isRequired,
};

ViolationsTablePanel.defaultProps = {
    selectedAlertId: undefined,
};

export default ViolationsTablePanel;
