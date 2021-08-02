import React, { useState, ReactElement } from 'react';
import {
    Flex,
    FlexItem,
    Divider,
    PageSection,
    Title,
    Badge,
    Pagination,
    Select,
    SelectOption,
} from '@patternfly/react-core';

import useTableSelection from 'hooks/useTableSelection';
import { TableColumn, SortDirection } from 'hooks/useTableSort';
import ViolationsTable from './ViolationsTable';
import { Violation } from './types/violationTypes';
// import dialogues from './dialogues';
// import ResolveConfirmation from './Dialogues/ResolveConfirmation';
// import ExcludeConfirmation from './Dialogues/ExcludeConfirmation';
// import TagConfirmation from './Dialogues/TagConfirmation';

type ViolationsTablePanelProps = {
    violations: Violation[];
    violationsCount: number;
    // selectedAlertId?: string;
    setSelectedAlertId: (id) => void;
    currentPage: number;
    setCurrentPage: (page) => void;
    resolvableAlerts: Set<string>;
    excludableAlerts: Violation[];
    perPage: number;
    setPerPage: (perPage) => void;
    activeSortIndex: number;
    setActiveSortIndex: (idx) => void;
    activeSortDirection: SortDirection;
    setActiveSortDirection: (dir) => void;
    columns: TableColumn[];
};

function ViolationsTablePanel({
    violations,
    violationsCount,
    // selectedAlertId,
    setSelectedAlertId,
    currentPage,
    setCurrentPage,
    perPage,
    setPerPage,
    resolvableAlerts,
    excludableAlerts,
    activeSortIndex,
    setActiveSortIndex,
    activeSortDirection,
    setActiveSortDirection,
    columns,
}: ViolationsTablePanelProps): ReactElement {
    // // Handle confirmation dialogue being open.
    // const [dialogue, setDialogue] = useState(null);

    // Handle Row Actions dropdown state.
    const [isSelectOpen, setIsSelectOpen] = useState(false);
    const {
        selected,
        allRowsSelected,
        hasSelections,
        onSelect,
        onSelectAll,
        getSelectedIds,
    } = useTableSelection(violations);

    function onToggleSelect(toggleOpen) {
        setIsSelectOpen(toggleOpen);
    }

    // // Handle dialogue pop ups.
    // function showResolveConfirmationDialog() {
    //     setDialogue(dialogues.resolve);
    // }
    // function showExcludeConfirmationDialog() {
    //     setDialogue(dialogues.excludeScopes);
    // }
    // function showTagConfirmationDialog() {
    //     setDialogue(dialogues.tag);
    // }

    // Handle page changes.
    function changePage(e, newPage) {
        if (newPage !== currentPage) {
            setCurrentPage(newPage);
        }
    }

    function changePerPage(e, newPerPage) {
        setPerPage(newPerPage);
    }

    const excludableAlertIds: Set<string> = new Set(excludableAlerts.map((alert) => alert.id));
    const selectedIds = getSelectedIds();
    const numSelected = selectedIds.length;
    let numResolveable = 0;
    let numScopesToExclude = 0;

    selectedIds.forEach((id) => {
        if (excludableAlertIds.has(id)) {
            numScopesToExclude += 1;
        }
        if (resolvableAlerts.has(id)) {
            numResolveable += 1;
        }
    });

    return (
        <>
            <Flex
                className="pf-u-p-md"
                alignSelf={{ default: 'alignSelfCenter' }}
                fullWidth={{ default: 'fullWidth' }}
            >
                <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                    <Title headingLevel="h2" className="pf-u-color-100 pf-u-ml-sm">
                        Violations
                    </Title>
                </FlexItem>
                <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                    <Badge isRead>{violationsCount}</Badge>
                </FlexItem>
                <FlexItem>
                    <Select
                        onToggle={onToggleSelect}
                        isOpen={isSelectOpen}
                        placeholderText="Row Actions"
                        // onSelect={null}
                        isDisabled={!hasSelections}
                    >
                        <SelectOption
                            key="0"
                            value={`Add Tags for Violations (${numSelected})`}
                            // onClick={showTagConfirmationDialog}
                        />
                        <SelectOption
                            key="1"
                            value={`Mark as Resolved (${numResolveable})`}
                            isDisabled={numResolveable === 0}
                            // onClick={showResolveConfirmationDialog}
                        />
                        <SelectOption
                            key="2"
                            value={`Exclude (${numScopesToExclude})`}
                            isDisabled={numScopesToExclude === 0}
                            // onClick={showExcludeConfirmationDialog}
                        />
                    </Select>
                </FlexItem>
                <FlexItem align={{ default: 'alignRight' }}>
                    <Pagination
                        itemCount={violationsCount}
                        page={currentPage}
                        onSetPage={changePage}
                        perPage={perPage}
                        onPerPageSelect={changePerPage}
                    />
                </FlexItem>
            </Flex>
            <Divider component="div" />
            <PageSection isFilled padding={{ default: 'noPadding' }} hasOverflowScroll>
                <ViolationsTable
                    violations={violations}
                    // selectedAlertId={selectedAlertId}
                    setSelectedAlertId={setSelectedAlertId}
                    selected={selected}
                    onSelect={onSelect}
                    onSelectAll={onSelectAll}
                    allRowsSelected={allRowsSelected}
                    activeSortIndex={activeSortIndex}
                    setActiveSortIndex={setActiveSortIndex}
                    activeSortDirection={activeSortDirection}
                    setActiveSortDirection={setActiveSortDirection}
                    columns={columns}
                />
            </PageSection>
            {/* {dialogue === dialogues.excludeScopes && (
                <ExcludeConfirmation
                    setDialogue={setDialogue}
                    excludableAlerts={excludableAlerts}
                    checkedAlertIds={selectedIds}
                    setCheckedAlertIds={onSelect}
                />
            )}
            {dialogue === dialogues.resolve && (
                <ResolveConfirmation
                    setDialogue={setDialogue}
                    checkedAlertIds={selectedIds}
                    setCheckedAlertIds={onSelect}
                    resolvableAlerts={resolvableAlerts}
                />
            )}
            {dialogue === dialogues.tag && (
                <TagConfirmation
                    setDialogue={setDialogue}
                    checkedAlertIds={selectedIds}
                    setCheckedAlertIds={onSelect}
                />
            )} */}
        </>
    );
}

export default ViolationsTablePanel;
