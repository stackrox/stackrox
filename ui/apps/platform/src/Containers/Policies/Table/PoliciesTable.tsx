import React, { useState, useEffect } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Button,
    ButtonVariant,
    Dropdown,
    DropdownItem,
    DropdownSeparator,
    DropdownToggle,
    Flex,
    PageSection,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
    Truncate,
} from '@patternfly/react-core';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import { CaretDownIcon, CheckCircleIcon } from '@patternfly/react-icons';
import orderBy from 'lodash/orderBy';
import pluralize from 'pluralize';

import { ListPolicy } from 'types/policy.proto';
import { sortSeverity, sortAsciiCaseInsensitive, sortValueByLength } from 'sorters/sorters';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import LinkShim from 'Components/PatternFly/LinkShim';
import SearchFilterInput from 'Components/SearchFilterInput';
import { ActionItem } from 'Containers/Violations/ViolationsTablePanel';
import EnableDisableNotificationModal, {
    EnableDisableType,
} from 'Containers/Policies/Modal/EnableDisableNotificationModal';
import useTableSelection from 'hooks/useTableSelection';
import { AlertVariantType } from 'hooks/patternfly/useToasts';
import { policiesBasePath } from 'routePaths';
import { NotifierIntegration } from 'types/notifier.proto';
import { SearchFilter } from 'types/search';
import { SortDirection } from 'types/table';

import {
    LabelAndNotifierIdsForType,
    formatLifecycleStages,
    formatNotifierCountsWithLabelStrings,
    getLabelAndNotifierIdsForTypes,
} from '../policies.utils';
import PolicySeverityLabel from '../PolicySeverityLabel';

import './PoliciesTable.css';

const columns = [
    {
        Header: 'Policy',
        accessor: 'name',
        sortMethod: (a: ListPolicy, b: ListPolicy) => sortAsciiCaseInsensitive(a.name, b.name),
        width: 20 as const,
    },
    {
        Header: 'Description',
        accessor: 'description',
        width: 40 as const,
    },
    {
        Header: 'Status',
        accessor: 'disabled',
        width: 15 as const,
    },
    {
        Header: 'Notifiers',
        accessor: 'notifiers',
        sortMethod: (a: ListPolicy, b: ListPolicy) => sortValueByLength(a.notifiers, b.notifiers),
    },
    {
        Header: 'Severity',
        accessor: 'severity',
        sortMethod: (a: ListPolicy, b: ListPolicy) => -sortSeverity(a.severity, b.severity),
    },
    {
        Header: 'Lifecycle',
        accessor: 'lifecycleStages',
    },
];

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
    searchFilter,
    searchOptions,
}: PoliciesTableProps): React.ReactElement {
    const history = useHistory();
    const [labelAndNotifierIdsForTypes, setLabelAndNotifierIdsForTypes] = useState<
        LabelAndNotifierIdsForType[]
    >([]);

    const [deletingIds, setDeletingIds] = useState<string[]>([]);
    const [isDeleting, setIsDeleting] = useState(false);

    const [enableDisableType, setEnableDisableType] = useState<EnableDisableType | null>(null);

    // index of the currently active column
    const [activeSortIndex, setActiveSortIndex] = useState(0);
    // sort direction of the currently active column
    const [activeSortDirection, setActiveSortDirection] = useState<SortDirection>('asc');
    // Handle Bulk Actions dropdown state.
    const [isActionsOpen, setIsActionsOpen] = useState(false);
    // For sorting data client side
    const [rows, setRows] = useState<ListPolicy[]>([]);

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
        // onClearAll,
        getSelectedIds,
    } = useTableSelection(policies);

    function onToggleActions(toggleOpen) {
        setIsActionsOpen(toggleOpen);
    }

    function onSelectActions() {
        setIsActionsOpen(false);
    }

    function onSort(e, index, direction) {
        setActiveSortIndex(index);
        setActiveSortDirection(direction);
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

    // If we use server side page management, this becomes unnecessary
    useEffect(() => {
        const { sortMethod, accessor } = columns[activeSortIndex];
        let sortedPolicies = [...policies];
        if (sortMethod) {
            sortedPolicies.sort(sortMethod);
            if (activeSortDirection === 'desc') {
                sortedPolicies.reverse();
            }
        } else {
            sortedPolicies = orderBy(sortedPolicies, [accessor], [activeSortDirection]);
        }
        setRows(sortedPolicies);
    }, [policies, activeSortIndex, activeSortDirection]);

    function onConfirmDeletePolicy() {
        setIsDeleting(true);
        deletePoliciesHandler(deletingIds).finally(() => {
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
                            className="pf-u-flex-grow-1 pf-u-flex-shrink-1"
                        >
                            <SearchFilterInput
                                className="w-full pf-search-shim"
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
                            <ToolbarItem>
                                <Dropdown
                                    data-testid="policies-bulk-actions-dropdown"
                                    onSelect={onSelectActions}
                                    toggle={
                                        <DropdownToggle
                                            isDisabled={!hasWriteAccessForPolicy || !hasSelections}
                                            isPrimary
                                            onToggle={onToggleActions}
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
                                        // TODO: https://stack-rox.atlassian.net/browse/ROX-8613
                                        // Export policies to JSON
                                        // onClick={() => exportPoliciesHandler(selectedIds, onClearAll)}
                                        // {`Export policies to JSON (${numSelected})`}
                                        <DropdownSeparator key="Separator" />,
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
                            <ToolbarItem>
                                <Tooltip content="Manually enrich external data">
                                    <Button variant="secondary" onClick={onClickReassessPolicies}>
                                        Reassess all
                                    </Button>
                                </Tooltip>
                            </ToolbarItem>
                        </ToolbarGroup>
                        <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                            <Pagination
                                isCompact
                                isDisabled
                                itemCount={rows.length}
                                page={1}
                                perPage={rows.length}
                            />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
                <TableComposable
                    isStickyHeader
                    aria-label="Policies table"
                    data-testid="policies-table"
                >
                    <Thead>
                        <Tr>
                            <Th
                                select={{
                                    onSelect: onSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            {columns.map(({ Header, width }, columnIndex) => {
                                const sortParams = {
                                    sort: {
                                        sortBy: {
                                            index: activeSortIndex,
                                            direction: activeSortDirection,
                                        },
                                        onSort,
                                        columnIndex,
                                    },
                                };
                                return (
                                    <Th key={Header} modifier="wrap" width={width} {...sortParams}>
                                        {Header}
                                    </Th>
                                );
                            })}
                            <Th />
                        </Tr>
                    </Thead>
                    <Tbody>
                        {rows.map((policy) => {
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
                            const notifierCountsWithLabelStrings =
                                formatNotifierCountsWithLabelStrings(
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
                                <Tr key={id}>
                                    <Td
                                        select={{
                                            rowIndex,
                                            onSelect,
                                            isSelected: selected[rowIndex],
                                        }}
                                    />
                                    <Td dataLabel="Policy">
                                        <Button
                                            variant={ButtonVariant.link}
                                            isInline
                                            component={LinkShim}
                                            href={`${policiesBasePath}/${id}`}
                                        >
                                            {name}
                                        </Button>
                                    </Td>
                                    <Td dataLabel="Description">
                                        <Truncate
                                            content={description || '-'}
                                            tooltipPosition="top"
                                        />
                                    </Td>
                                    <Td dataLabel="Status">
                                        {disabled ? (
                                            'Disabled'
                                        ) : (
                                            <Flex className="pf-u-info-color-200">
                                                <CheckCircleIcon className="pf-u-mr-sm pf-m-align-self-center" />
                                                Enabled
                                            </Flex>
                                        )}
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
                                                            className="pf-u-text-nowrap"
                                                        >
                                                            {notifierCountWithLabelString}
                                                        </div>
                                                    )
                                                )}
                                            </>
                                        )}
                                    </Td>
                                    <Td dataLabel="Severity">
                                        <PolicySeverityLabel severity={severity} />
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
                            );
                        })}
                    </Tbody>
                </TableComposable>
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
