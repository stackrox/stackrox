import React, { useState, useEffect } from 'react';
import { Link, useHistory } from 'react-router-dom';
import {
    Button,
    PageSection,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import {
    Dropdown,
    DropdownItem,
    DropdownSeparator,
    DropdownToggle,
} from '@patternfly/react-core/deprecated';
import { Table, Thead, Tbody, Tr, Th, Td, ExpandableRowContent } from '@patternfly/react-table';
import { CaretDownIcon } from '@patternfly/react-icons';
import pluralize from 'pluralize';

import { ListPolicy } from 'types/policy.proto';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import PolicyDisabledIconText from 'Components/PatternFly/IconText/PolicyDisabledIconText';
import PolicySeverityIconText from 'Components/PatternFly/IconText/PolicySeverityIconText';
import SearchFilterInput from 'Components/SearchFilterInput';
import { ActionItem } from 'Containers/Violations/ViolationsTablePanel';
import EnableDisableNotificationModal, {
    EnableDisableType,
} from 'Containers/Policies/Modal/EnableDisableNotificationModal';
import useTableSelection from 'hooks/useTableSelection';
import useSet from 'hooks/useSet';
import { AlertVariantType } from 'hooks/patternfly/useToasts';
import { UseURLSortResult } from 'hooks/useURLSort';
import { policiesBasePath } from 'routePaths';
import { NotifierIntegration } from 'types/notifier.proto';
import { SearchFilter } from 'types/search';
import { columns, defaultPolicyLabel, userPolicyLabel } from './PoliciesTable.utils';
import {
    LabelAndNotifierIdsForType,
    formatLifecycleStages,
    formatNotifierCountsWithLabelStrings,
    getLabelAndNotifierIdsForTypes,
} from '../policies.utils';

import './PoliciesTable.css';

type PoliciesTableProps = {
    notifiers: NotifierIntegration[];
    policies?: ListPolicy[];
    fetchPoliciesHandler: () => void;
    addToast: (text: string, variant: AlertVariantType, content?: string) => void;
    hasWriteAccessForPolicy: boolean;
    deletePoliciesHandler: (ids: string[]) => Promise<void>;
    exportPoliciesHandler: (ids, onClearAll?) => void;
    enablePoliciesHandler: (ids) => void;
    disablePoliciesHandler: (ids) => void;
    handleChangeSearchFilter: (searchFilter: SearchFilter) => void;
    onClickReassessPolicies: () => void;
    getSortParams: UseURLSortResult['getSortParams'];
    searchFilter?: SearchFilter;
    searchOptions: string[];
};

function PoliciesTable({
    notifiers,
    policies = [],
    fetchPoliciesHandler,
    addToast,
    hasWriteAccessForPolicy,
    deletePoliciesHandler,
    exportPoliciesHandler,
    enablePoliciesHandler,
    disablePoliciesHandler,
    handleChangeSearchFilter,
    onClickReassessPolicies,
    getSortParams,
    searchFilter,
    searchOptions,
}: PoliciesTableProps): React.ReactElement {
    const expandedRowSet = useSet<string>();
    const history = useHistory();
    const [labelAndNotifierIdsForTypes, setLabelAndNotifierIdsForTypes] = useState<
        LabelAndNotifierIdsForType[]
    >([]);

    const [deletingIds, setDeletingIds] = useState<string[]>([]);
    const [isDeleting, setIsDeleting] = useState(false);

    const [enableDisableType, setEnableDisableType] = useState<EnableDisableType | null>(null);

    // Handle Bulk Actions dropdown state.
    const [isActionsOpen, setIsActionsOpen] = useState(false);
    // For sorting data client side

    useEffect(() => {
        setLabelAndNotifierIdsForTypes(getLabelAndNotifierIdsForTypes(notifiers));
    }, [notifiers]);

    // a map to keep track of row index within the table to the policy id
    // for checkbox selection after the table has been sorted
    const rowIdToIndex = {};
    policies.forEach(({ id }, idx) => {
        rowIdToIndex[id] = idx;
    });

    // Handle selected rows in table
    const {
        selected,
        allRowsSelected,
        hasSelections,
        onSelect,
        onSelectAll,
        onClearAll,
        getSelectedIds,
    } = useTableSelection(policies);

    function onToggleActions(toggleOpen) {
        setIsActionsOpen(toggleOpen);
    }

    function onSelectActions() {
        setIsActionsOpen(false);
    }

    function onEditPolicy(id: string) {
        history.push({
            pathname: `${policiesBasePath}/${id}`,
            search: 'action=edit',
        });
    }

    function onClonePolicy(id: string) {
        history.push({
            pathname: `${policiesBasePath}/${id}`,
            search: 'action=clone',
        });
    }

    const selectedIds = getSelectedIds();
    const selectedPolicies = policies.filter(({ id }) => selectedIds.includes(id));
    let numEnabled = 0;
    let numDisabled = 0;
    let numDeletable = 0;
    selectedPolicies.forEach(({ disabled, isDefault }) => {
        if (disabled) {
            numDisabled += 1;
        } else {
            numEnabled += 1;
        }
        if (!isDefault) {
            numDeletable += 1;
        }
    });

    function onConfirmDeletePolicy() {
        setIsDeleting(true);
        deletePoliciesHandler(deletingIds)
            .catch(() => {
                // TODO render error in dialog and move finally code to then block.
            })
            .finally(() => {
                setDeletingIds([]);
                setIsDeleting(false);
            });
    }

    function onCancelDeletePolicy() {
        setDeletingIds([]);
    }

    // TODO: https://stack-rox.atlassian.net/browse/ROX-8613
    // isDisabled={!hasSelections}
    // dropdownItems={hasWriteAccessForPolicy ? [Enable, Disable, Export, Delete] : [Export]} see PolicyDetail.tsx
    return (
        <>
            <PageSection isFilled id="policies-table">
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem
                            variant="search-filter"
                            className="pf-v5-u-flex-grow-1 pf-v5-u-flex-shrink-1"
                        >
                            <SearchFilterInput
                                className="w-full theme-light pf-search-shim"
                                handleChangeSearchFilter={handleChangeSearchFilter}
                                placeholder="Filter policies"
                                searchCategory="POLICIES"
                                searchFilter={searchFilter ?? {}}
                                searchOptions={searchOptions}
                            />
                        </ToolbarItem>
                        <ToolbarGroup
                            spaceItems={{ default: 'spaceItemsSm' }}
                            variant="button-group"
                        >
                            {hasWriteAccessForPolicy && (
                                <ToolbarItem>
                                    <Dropdown
                                        data-testid="policies-bulk-actions-dropdown"
                                        onSelect={onSelectActions}
                                        toggle={
                                            <DropdownToggle
                                                isDisabled={!hasSelections}
                                                toggleVariant="primary"
                                                onToggle={(_event, toggleOpen) =>
                                                    onToggleActions(toggleOpen)
                                                }
                                                toggleIndicator={CaretDownIcon}
                                            >
                                                Bulk actions
                                            </DropdownToggle>
                                        }
                                        isOpen={isActionsOpen}
                                        dropdownItems={[
                                            <DropdownItem
                                                key="Enable policies"
                                                component="button"
                                                isDisabled={numDisabled === 0}
                                                onClick={() => enablePoliciesHandler(selectedIds)}
                                            >
                                                {`Enable policies (${numDisabled})`}
                                            </DropdownItem>,
                                            <DropdownItem
                                                key="Disable policies"
                                                component="button"
                                                isDisabled={numEnabled === 0}
                                                onClick={() => disablePoliciesHandler(selectedIds)}
                                            >
                                                {`Disable policies (${numEnabled})`}
                                            </DropdownItem>,
                                            <DropdownSeparator key="Separator-1" />,
                                            <DropdownItem
                                                key="Enable notification"
                                                component="button"
                                                onClick={() => {
                                                    setEnableDisableType('enable');
                                                }}
                                            >
                                                Enable notification
                                            </DropdownItem>,
                                            <DropdownItem
                                                key="Disable notification"
                                                component="button"
                                                onClick={() => {
                                                    setEnableDisableType('disable');
                                                }}
                                            >
                                                Disable notification
                                            </DropdownItem>,
                                            <DropdownSeparator key="Separator-2" />,
                                            <DropdownItem
                                                key="Export policy"
                                                component="button"
                                                isDisabled={selectedPolicies.length === 0}
                                                onClick={() =>
                                                    exportPoliciesHandler(selectedIds, onClearAll)
                                                }
                                            >
                                                {`Export policies (${selectedPolicies.length})`}
                                            </DropdownItem>,
                                            <DropdownSeparator key="Separator" />,
                                            <DropdownItem
                                                key="Delete policy"
                                                component="button"
                                                isDisabled={numDeletable === 0}
                                                onClick={() =>
                                                    setDeletingIds(
                                                        selectedPolicies
                                                            .filter(({ isDefault }) => !isDefault)
                                                            .map(({ id }) => id)
                                                    )
                                                }
                                            >
                                                {`Delete policies (${numDeletable})`}
                                            </DropdownItem>,
                                        ]}
                                    />
                                </ToolbarItem>
                            )}
                            <ToolbarItem>
                                <Tooltip content="Manually enrich external data">
                                    <Button variant="secondary" onClick={onClickReassessPolicies}>
                                        Reassess all
                                    </Button>
                                </Tooltip>
                            </ToolbarItem>
                        </ToolbarGroup>
                        <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                            <Pagination
                                isCompact
                                isDisabled
                                itemCount={policies.length}
                                page={1}
                                perPage={policies.length}
                            />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
                <Table isStickyHeader aria-label="Policies table" data-testid="policies-table">
                    <Thead>
                        <Tr>
                            <Th>{/* Header for expanded column */}</Th>
                            <Th
                                select={{
                                    onSelect: onSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            {columns.map(({ Header, width }) => {
                                // const sortParams = {
                                //     sort: {
                                //         sortBy: {
                                //             index: activeSortIndex,
                                //             direction: activeSortDirection,
                                //         },
                                //         onSort,
                                //         columnIndex,
                                //     },
                                // };
                                return (
                                    <Th
                                        key={Header}
                                        modifier="wrap"
                                        width={width}
                                        sort={getSortParams(Header)}
                                    >
                                        {Header}
                                    </Th>
                                );
                            })}
                            <Td />
                        </Tr>
                    </Thead>
                    {policies.map((policy) => {
                        const {
                            description,
                            disabled,
                            id,
                            isDefault,
                            lifecycleStages,
                            name,
                            notifiers: notifierIds,
                            severity,
                        } = policy;
                        const isExpanded = expandedRowSet.has(id);

                        const notifierCountsWithLabelStrings = formatNotifierCountsWithLabelStrings(
                            labelAndNotifierIdsForTypes,
                            notifierIds
                        );
                        const exportPolicyAction: ActionItem = {
                            title: 'Export policy to JSON',
                            onClick: () => exportPoliciesHandler([id]),
                        };
                        const actionItems = hasWriteAccessForPolicy
                            ? [
                                  {
                                      title: 'Edit policy',
                                      onClick: () => onEditPolicy(id),
                                  },
                                  {
                                      title: 'Clone policy',
                                      onClick: () => onClonePolicy(id),
                                  },
                                  disabled
                                      ? {
                                            title: 'Enable policy',
                                            onClick: () => enablePoliciesHandler([id]),
                                        }
                                      : {
                                            title: 'Disable policy',
                                            onClick: () => disablePoliciesHandler([id]),
                                        },
                                  exportPolicyAction,
                                  {
                                      isSeparator: true,
                                  },
                                  {
                                      title: isDefault
                                          ? 'Cannot delete a default policy'
                                          : 'Delete policy',
                                      onClick: () => setDeletingIds([id]),
                                      isDisabled: isDefault,
                                  },
                              ]
                            : [exportPolicyAction];
                        const rowIndex = rowIdToIndex[id];
                        return (
                            <Tbody
                                key={id}
                                style={{
                                    borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                                }}
                                isExpanded={isExpanded}
                            >
                                <Tr>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () => expandedRowSet.toggle(id),
                                        }}
                                    />
                                    <Td
                                        select={{
                                            rowIndex,
                                            onSelect,
                                            isSelected: selected[rowIndex],
                                        }}
                                    />
                                    <Td dataLabel="Policy">
                                        <Link to={`${policiesBasePath}/${id}`}>{name}</Link>
                                    </Td>
                                    <Td dataLabel="Status">
                                        <PolicyDisabledIconText isDisabled={disabled} />
                                    </Td>
                                    <Td dataLabel="Origin">
                                        {isDefault ? defaultPolicyLabel : userPolicyLabel}
                                    </Td>
                                    <Td dataLabel="Notifiers">
                                        {notifierCountsWithLabelStrings.length === 0 ? (
                                            '-'
                                        ) : (
                                            <>
                                                {notifierCountsWithLabelStrings.map(
                                                    (notifierCountWithLabelString) => (
                                                        <div
                                                            key={notifierCountWithLabelString}
                                                            className="pf-v5-u-text-nowrap"
                                                        >
                                                            {notifierCountWithLabelString}
                                                        </div>
                                                    )
                                                )}
                                            </>
                                        )}
                                    </Td>
                                    <Td dataLabel="Severity">
                                        <PolicySeverityIconText severity={severity} />
                                    </Td>
                                    <Td dataLabel="Lifecycle">
                                        {formatLifecycleStages(lifecycleStages)}
                                    </Td>
                                    <Td
                                        actions={{
                                            items: actionItems,
                                        }}
                                    />
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td />
                                    <Td colSpan={6}>
                                        <ExpandableRowContent>{description}</ExpandableRowContent>
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })}
                </Table>
            </PageSection>
            <ConfirmationModal
                ariaLabel="Confirm delete"
                confirmText="Delete"
                isLoading={isDeleting}
                isOpen={deletingIds.length !== 0}
                onConfirm={onConfirmDeletePolicy}
                onCancel={onCancelDeletePolicy}
            >
                Are you sure you want to delete {deletingIds.length}&nbsp;
                {pluralize('policy', deletingIds.length)}?
            </ConfirmationModal>
            <EnableDisableNotificationModal
                enableDisableType={enableDisableType}
                setEnableDisableType={setEnableDisableType}
                fetchPoliciesHandler={fetchPoliciesHandler}
                addToast={addToast}
                selectedPolicyIds={selectedIds}
                notifiers={notifiers}
            />
        </>
    );
}

export default PoliciesTable;
