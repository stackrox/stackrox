import React, { useState, ReactElement } from 'react';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    Divider,
    Title,
    PageSection,
    Pagination,
    pluralize,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';
import { ActionsColumn, Table, Tbody, Thead, Td, Th, Tr } from '@patternfly/react-table';

import { ENFORCEMENT_ACTIONS } from 'constants/enforcementActions';
import { VIOLATION_STATES } from 'constants/violationStates';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import TableCell from 'Components/PatternFly/TableCell';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';
import useTableSelection from 'hooks/useTableSelection';
import { GetSortParams } from 'hooks/useURLSort';
import useRestMutation from 'hooks/useRestMutation';
import useToasts from 'hooks/patternfly/useToasts';
import { resolveAlert } from 'services/AlertsService';
import { excludeDeployments } from 'services/PoliciesService';
import { ListAlert } from 'types/alert.proto';
import { TableColumn } from 'types/table';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { SearchFilter } from 'types/search';
import { OnSearchCallback } from 'Components/CompoundSearchFilter/types';
import ResolveConfirmation from './Modals/ResolveConfirmation';
import ExcludeConfirmation from './Modals/ExcludeConfirmation';
import ViolationsTableSearchFilter from './ViolationsTableSearchFilter';

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
    searchFilter: SearchFilter;
    onFilterChange: (newFilter: SearchFilter) => void;
    onSearch: OnSearchCallback;
    additionalContextFilter: SearchFilter;
    hasActiveViolations: boolean;
    isTableDataUpdating: boolean;
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
    searchFilter,
    onFilterChange,
    onSearch,
    additionalContextFilter,
    hasActiveViolations,
    isTableDataUpdating,
}: ViolationsTablePanelProps): ReactElement {
    const isRouteEnabled = useIsRouteEnabled();
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForAlert = hasReadWriteAccess('Alert');
    // Require READ_WRITE_ACCESS to exclude plus READ_ACCESS to other resources for Policies route.
    const hasWriteAccessForExcludeDeploymentsFromPolicy =
        hasReadWriteAccess('WorkflowAdministration') && isRouteEnabled('policy-management');
    const hasActions =
        hasActiveViolations &&
        (hasWriteAccessForAlert || hasWriteAccessForExcludeDeploymentsFromPolicy);

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

    const { toasts, addToast, removeToast } = useToasts();

    const excludeDeploymentMutation = useRestMutation(
        ({ policyId, deploymentNames }: { policyId: string; deploymentNames: string[] }) =>
            excludeDeployments(policyId, deploymentNames),
        {
            onSuccess: () => {
                addToast('Deployment excluded from policy', 'success');
            },
            onError: (err: unknown) => {
                addToast(
                    'There was an error excluding the deployment',
                    'danger',
                    getAxiosErrorMessage(err)
                );
            },
        }
    );

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
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }) => (
                    <Alert
                        key={key}
                        variant={variant}
                        title={title}
                        component="p"
                        timeout={variant === 'success'}
                        onTimeout={() => removeToast(key)}
                        actionClose={
                            <AlertActionCloseButton
                                title={title}
                                variantLabel={variant}
                                onClose={() => removeToast(key)}
                            />
                        }
                    >
                        {children}
                    </Alert>
                ))}
            </AlertGroup>
            <ViolationsTableSearchFilter
                searchFilter={searchFilter}
                onFilterChange={onFilterChange}
                onSearch={onSearch}
                additionalContextFilter={additionalContextFilter}
            />
            <Divider component="div" />
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem>
                        <Title headingLevel="h2" className="pf-v5-u-color-100">
                            {pluralize(violationsCount, 'result')} found
                        </Title>
                    </ToolbarItem>
                    {hasActions && (
                        <ToolbarItem align={{ default: 'alignRight' }}>
                            <Select
                                onToggle={(_event, toggleOpen) => onToggleSelect(toggleOpen)}
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
                        </ToolbarItem>
                    )}
                    <ToolbarItem align={{ default: 'alignRight' }} variant="pagination">
                        <Pagination
                            itemCount={violationsCount}
                            page={currentPage}
                            onSetPage={changePage}
                            perPage={perPage}
                            onPerPageSelect={changePerPage}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider component="div" />
            <PageSection isFilled padding={{ default: 'noPadding' }} hasOverflowScroll>
                <Table variant="compact" isStickyHeader>
                    <Thead>
                        <Tr>
                            {hasActions && (
                                <Th
                                    select={{
                                        onSelect: onSelectAll,
                                        isSelected: allRowsSelected,
                                    }}
                                />
                            )}
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
                            {hasActions && (
                                <Th>
                                    <span className="pf-v5-screen-reader">Row actions</span>
                                </Th>
                            )}
                        </Tr>
                    </Thead>
                    <Tbody aria-live="polite" aria-busy={isTableDataUpdating ? 'true' : 'false'}>
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
                                        excludeDeploymentMutation.mutate({
                                            policyId: policy.id,
                                            deploymentNames: [violation.deployment.name],
                                        }),
                                });
                            }
                            return (
                                <Tr key={id}>
                                    {hasActions && (
                                        <Td
                                            select={{
                                                rowIndex,
                                                onSelect,
                                                isSelected: selected[rowIndex],
                                            }}
                                        />
                                    )}
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
                                        <Td isActionCell>
                                            <ActionsColumn
                                                // menuAppendTo={() => document.body}
                                                isDisabled={actionItems.length === 0}
                                                items={actionItems}
                                            />
                                        </Td>
                                    )}
                                </Tr>
                            );
                        })}
                    </Tbody>
                </Table>
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
