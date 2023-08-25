import React, { useState, ReactElement } from 'react';
import {
    Flex,
    FlexItem,
    Divider,
    PageSection,
    Title,
    Pagination,
    Select,
    SelectOption,
    pluralize,
} from '@patternfly/react-core';
import { ActionsColumn, TableComposable, Tbody, Thead, Td, Th, Tr } from '@patternfly/react-table';

import { ENFORCEMENT_ACTIONS } from 'constants/enforcementActions';
import VIOLATION_STATES from 'constants/violationStates';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import TableCell from 'Components/PatternFly/TableCell';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';
import useTableSelection from 'hooks/useTableSelection';
import { GetSortParams } from 'hooks/useURLSort';
import { resolveAlert } from 'services/AlertsService';
import { excludeDeployments } from 'services/PoliciesService';
import { ListAlert } from 'types/alert.proto';
import { TableColumn } from 'types/table';

import ResolveConfirmation from './Modals/ResolveConfirmation';
import ExcludeConfirmation from './Modals/ExcludeConfirmation';

export type ActionItem = {
    title: string | ReactElement;
    onClick: (item) => void;
};

type ModalType = 'resolve' | 'excludeScopes' | 'tag';

type ViolationsTablePanelProps = {
    violations: ListAlert[];
    violationsCount: number;
    currentPage: number;
    setCurrentPage: (page) => void;
    resolvableAlerts: Set<string>;
    excludableAlerts: ListAlert[];
    perPage: number;
    setPerPage: (perPage) => void;
    getSortParams: GetSortParams;
    columns: TableColumn[];
};

function ViolationsTablePanel({
    violations,
    violationsCount,
    currentPage,
    setCurrentPage,
    perPage,
    setPerPage,
    resolvableAlerts,
    excludableAlerts,
    getSortParams,
    columns,
}: ViolationsTablePanelProps): ReactElement {
    const isRouteEnabled = useIsRouteEnabled();
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForAlert = hasReadWriteAccess('Alert');
    // Require READ_WRITE_ACCESS to exclude plus READ_ACCESS to other resources for Policies route.
    const hasWriteAccessForExcludeDeploymentsFromPolicy =
        hasReadWriteAccess('WorkflowAdministration') && isRouteEnabled('policy-management');
    const hasActions = hasWriteAccessForAlert || hasWriteAccessForExcludeDeploymentsFromPolicy;

    // Handle confirmation modal being open.
    const [modalType, setModalType] = useState<ModalType>();

    // Handle Row Actions dropdown state.
    const [isSelectOpen, setIsSelectOpen] = useState(false);
    const {
        selected,
        allRowsSelected,
        hasSelections,
        onSelect,
        onSelectAll,
        onClearAll,
        getSelectedIds,
    } = useTableSelection(violations);

    function onToggleSelect(toggleOpen) {
        setIsSelectOpen(toggleOpen);
    }

    // Handle setting confirmation modals for bulk actions
    function showResolveConfirmationDialog() {
        setModalType('resolve');
    }
    function showExcludeConfirmationDialog() {
        setModalType('excludeScopes');
    }

    // Handle closing confirmation modals for bulk actions;
    function cancelModal() {
        setModalType(undefined);
    }

    // Handle closing confirmation modal and clearing selection;
    function closeModal() {
        setModalType(undefined);
        onClearAll();
    }

    // Handle page changes.
    function changePage(e, newPage) {
        if (newPage !== currentPage) {
            setCurrentPage(newPage);
        }
    }

    function changePerPage(e, newPerPage) {
        setPerPage(newPerPage);
    }

    function closeSelect() {
        setIsSelectOpen(false);
    }

    function resolveAlertAction(addToBaseline, id) {
        return resolveAlert(id, addToBaseline).then(onClearAll, onClearAll);
    }

    const excludableAlertIds: Set<string> = new Set(excludableAlerts.map((alert) => alert.id));
    const selectedIds = getSelectedIds();
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
                className="pf-u-pb-md"
                alignSelf={{ default: 'alignSelfCenter' }}
                fullWidth={{ default: 'fullWidth' }}
            >
                <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                    <Title headingLevel="h2" className="pf-u-color-100">
                        {pluralize(violationsCount, 'result')} found
                    </Title>
                </FlexItem>
                {hasActions && (
                    <FlexItem>
                        <Select
                            onToggle={onToggleSelect}
                            isOpen={isSelectOpen}
                            placeholderText="Row actions"
                            onSelect={closeSelect}
                            isDisabled={!hasSelections}
                        >
                            <SelectOption
                                key="1"
                                value={`Mark as resolved (${numResolveable})`}
                                isDisabled={!hasWriteAccessForAlert || numResolveable === 0}
                                onClick={showResolveConfirmationDialog}
                            />
                            <SelectOption
                                key="2"
                                value={`Exclude deployments from policy (${numScopesToExclude})`}
                                isDisabled={
                                    !hasWriteAccessForExcludeDeploymentsFromPolicy ||
                                    numScopesToExclude === 0
                                }
                                onClick={showExcludeConfirmationDialog}
                            />
                        </Select>
                    </FlexItem>
                )}
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
                <TableComposable variant="compact" isStickyHeader>
                    <Thead>
                        <Tr>
                            <Th
                                select={{
                                    onSelect: onSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            {columns.map(({ Header, sortField }) => {
                                const sortParams = sortField
                                    ? { sort: getSortParams(sortField) }
                                    : {};
                                return (
                                    <Th key={Header} modifier="wrap" {...sortParams}>
                                        {Header}
                                    </Th>
                                );
                            })}
                            {hasActions && <Td />}
                        </Tr>
                    </Thead>
                    <Tbody>
                        {violations.map((violation, rowIndex) => {
                            const { state, lifecycleStage, enforcementAction, policy, id } =
                                violation;
                            const isAttemptedViolation = state === VIOLATION_STATES.ATTEMPTED;
                            const isResolved = state === VIOLATION_STATES.RESOLVED;
                            const isRuntimeAlert = lifecycleStage === LIFECYCLE_STAGES.RUNTIME;
                            const isDeployCreateAttemptedAlert =
                                enforcementAction ===
                                ENFORCEMENT_ACTIONS.FAIL_DEPLOYMENT_CREATE_ENFORCEMENT;

                            // Instead of items prop of Td element, render ActionsColumn element
                            // so every cell has vertical ellipsis (also known as kabob)
                            // even if its items array is empty. For example:
                            // hasWriteAccessForAlert but alert is not Runtime lifecycle.
                            // !hasWriteAccessForWorkflowAdministration
                            const actionItems: ActionItem[] = [];
                            if (hasWriteAccessForAlert && !isResolved) {
                                if (isRuntimeAlert) {
                                    actionItems.push({
                                        title: 'Resolve and add to process baseline',
                                        onClick: () => resolveAlertAction(true, violation.id),
                                    });
                                }
                                if (isRuntimeAlert || isAttemptedViolation) {
                                    actionItems.push({
                                        title: 'Mark as resolved',
                                        onClick: () => resolveAlertAction(false, violation.id),
                                    });
                                }
                            }
                            if (
                                hasWriteAccessForExcludeDeploymentsFromPolicy &&
                                !isDeployCreateAttemptedAlert &&
                                'deployment' in violation
                            ) {
                                actionItems.push({
                                    title: 'Exclude deployment from policy',
                                    onClick: () =>
                                        excludeDeployments(policy.id, [violation.deployment.name]),
                                });
                            }
                            return (
                                // eslint-disable-next-line react/no-array-index-key
                                <Tr key={rowIndex}>
                                    <Td
                                        key={id}
                                        select={{
                                            rowIndex,
                                            onSelect,
                                            isSelected: selected[rowIndex],
                                        }}
                                    />
                                    {columns.map((column) => {
                                        return (
                                            <TableCell
                                                key={column.Header}
                                                row={violation}
                                                column={column}
                                            />
                                        );
                                    })}
                                    {hasActions && (
                                        <Td>
                                            <ActionsColumn
                                                isDisabled={actionItems.length === 0}
                                                items={actionItems}
                                            />
                                        </Td>
                                    )}
                                </Tr>
                            );
                        })}
                    </Tbody>
                </TableComposable>
            </PageSection>
            <ExcludeConfirmation
                isOpen={modalType === 'excludeScopes'}
                excludableAlerts={excludableAlerts}
                selectedAlertIds={selectedIds}
                closeModal={closeModal}
                cancelModal={cancelModal}
            />
            <ResolveConfirmation
                isOpen={modalType === 'resolve'}
                selectedAlertIds={selectedIds}
                resolvableAlerts={resolvableAlerts}
                closeModal={closeModal}
                cancelModal={cancelModal}
            />
        </>
    );
}

export default ViolationsTablePanel;
