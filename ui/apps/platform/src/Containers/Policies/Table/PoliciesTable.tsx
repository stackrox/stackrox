import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
    Button,
    Flex,
    FlexItem,
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
import {
    ActionsColumn,
    ExpandableRowContent,
    IAction,
    Table,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { CaretDownIcon } from '@patternfly/react-icons';

import { ListPolicy } from 'types/policy.proto';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import PolicyDisabledIconText from 'Components/PatternFly/IconText/PolicyDisabledIconText';
import PolicySeverityIconText from 'Components/PatternFly/IconText/PolicySeverityIconText';
import SearchFilterInput from 'Components/SearchFilterInput';
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
import {
    LabelAndNotifierIdsForType,
    formatLifecycleStages,
    formatNotifierCountsWithLabelStrings,
    getLabelAndNotifierIdsForTypes,
    getPolicyOriginLabel,
    isExternalPolicy,
} from '../policies.utils';

import './PoliciesTable.css';

function isExternalPolicySelected(policies: ListPolicy[], selectedIds: string[]): boolean {
    return policies.filter(({ id }) => selectedIds.includes(id)).some(isExternalPolicy);
}

type PoliciesTableProps = {
    notifiers: NotifierIntegration[];
    policies?: ListPolicy[];
    fetchPoliciesHandler: () => void;
    addToast: (text: string, variant: AlertVariantType, content?: string) => void;
    hasWriteAccessForPolicy: boolean;
    deletePoliciesHandler: (ids: string[]) => Promise<void>;
    exportPoliciesHandler: (ids, onClearAll?) => void;
    saveAsCustomResourceHandler: (ids, onClearAll?) => Promise<void>;
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
    saveAsCustomResourceHandler,
    enablePoliciesHandler,
    disablePoliciesHandler,
    handleChangeSearchFilter,
    onClickReassessPolicies,
    getSortParams,
    searchFilter,
    searchOptions,
}: PoliciesTableProps): React.ReactElement {
    const expandedRowSet = useSet<string>();
    const navigate = useNavigate();
    const [labelAndNotifierIdsForTypes, setLabelAndNotifierIdsForTypes] = useState<
        LabelAndNotifierIdsForType[]
    >([]);

    const [deletingIds, setDeletingIds] = useState<string[]>([]);
    const [isDeleting, setIsDeleting] = useState(false);

    const [savingIds, setSavingIds] = useState<string[]>([]);
    const [isSaving, setIsSaving] = useState(false);

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
        navigate(`${policiesBasePath}/${id}?action=edit`);
    }

    function onClonePolicy(id: string) {
        navigate(`${policiesBasePath}/${id}?action=clone`);
    }

    const selectedIds = getSelectedIds();
    const selectedPolicies = policies.filter(({ id }) => selectedIds.includes(id));
    let numEnabled = 0;
    let numDisabled = 0;
    let numDeletable = 0;
    let numSaveable = 0;
    selectedPolicies.forEach(({ disabled, isDefault }) => {
        if (disabled) {
            numDisabled += 1;
        } else {
            numEnabled += 1;
        }
        if (!isDefault) {
            numDeletable += 1;
            numSaveable += 1;
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

    function onConfirmSavePolicyAsCustomResource() {
        setIsSaving(true);
        saveAsCustomResourceHandler(savingIds)
            .catch(() => {
                // TODO render error in dialog and move finally code to then block.
            })
            .finally(() => {
                setSavingIds([]);
                setIsSaving(false);
            });
    }

    function onCancelSavePolicyAsCustomResource() {
        setSavingIds([]);
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
                                            <DropdownItem
                                                key="Save as Custom Resource"
                                                component="button"
                                                isDisabled={numSaveable === 0}
                                                onClick={() =>
                                                    setSavingIds(
                                                        selectedPolicies
                                                            .filter(({ isDefault }) => !isDefault)
                                                            .map(({ id }) => id)
                                                    )
                                                }
                                            >
                                                {`Save as Custom Resources (${numSaveable})`}
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
                            <Th>
                                <span className="pf-v5-screen-reader">Row expansion</span>
                            </Th>
                            <Th
                                select={{
                                    onSelect: onSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            {/* columns.map(({ Header, width }) => {
                                // https://github.com/stackrox/stackrox/pull/10316
                                // Move client-side sorting from PoliciesTable to PoliciesTablePage.
                                // After the Policies API is paginated in the API,
                                // we can use start passing the URL sort parameters that I (Van) have added here directly to the API call.
                                //
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
                            }) */}
                            <Th modifier="wrap" sort={getSortParams('Policy')} width={30}>
                                Policy
                            </Th>
                            <Th modifier="wrap" sort={getSortParams('Status')}>
                                Status
                            </Th>
                            <Th modifier="wrap" sort={getSortParams('Origin')} width={20}>
                                Origin
                            </Th>
                            <Th modifier="wrap" sort={getSortParams('Notifiers')}>
                                Notifiers
                            </Th>
                            <Th modifier="wrap" sort={getSortParams('Severity')}>
                                Severity
                            </Th>
                            <Th modifier="wrap" sort={getSortParams('Lifecycle')}>
                                Lifecycle
                            </Th>
                            <Th>
                                <span className="pf-v5-screen-reader">Row actions</span>
                            </Th>
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
                        const exportPolicyAction: IAction = {
                            title: 'Export policy to JSON',
                            onClick: () => exportPoliciesHandler([id]),
                        };
                        // Store as an array so that we can conditionally spread into actionItems
                        // based on feature flag without having to deal with nulls
                        const saveAsCustomResourceActionItem: IAction = {
                            title: isDefault
                                ? 'Cannot save as Custom Resource'
                                : 'Save as Custom Resource',
                            description: isDefault
                                ? 'Default policies cannot be saved as Custom Resource'
                                : '',
                            onClick: () => setSavingIds([id]),
                            isDisabled: isDefault,
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
                                  saveAsCustomResourceActionItem,
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
                            : [exportPolicyAction, saveAsCustomResourceActionItem];
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
                                    <Td dataLabel="Origin">{getPolicyOriginLabel(policy)}</Td>
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
                                    <Td isActionCell>
                                        <ActionsColumn items={actionItems} />
                                    </Td>
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
                title={`Delete policies? (${deletingIds.length})`}
                ariaLabel="Confirm delete"
                confirmText="Delete"
                isLoading={isDeleting}
                isOpen={deletingIds.length !== 0}
                onConfirm={onConfirmDeletePolicy}
                onCancel={onCancelDeletePolicy}
            >
                {isExternalPolicySelected(policies, deletingIds) ? (
                    <Flex direction={{ default: 'column' }}>
                        <FlexItem>
                            Deleted policies will be removed from the system and will no longer
                            trigger violations.
                        </FlexItem>
                        <FlexItem>
                            <em>
                                The current selection includes one or more externally managed
                                policies that will only be removed from the system temporarily until
                                the next resync. Locally managed policies will be removed
                                permanently.
                            </em>
                        </FlexItem>
                    </Flex>
                ) : (
                    <>
                        Deleted policies will be permanently removed from the system and will no
                        longer trigger violations.
                    </>
                )}
            </ConfirmationModal>
            <ConfirmationModal
                title={`Save policies as Custom Resources? (${savingIds.length})`}
                ariaLabel="Save as Custom Resources"
                confirmText="Yes"
                isLoading={isSaving}
                isOpen={savingIds.length !== 0}
                onConfirm={onConfirmSavePolicyAsCustomResource}
                onCancel={onCancelSavePolicyAsCustomResource}
                isDestructive={false}
            >
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <FlexItem>
                        Clicking <strong>Yes</strong> will save the policy as a Kubernetes custom
                        resource (YAML).
                    </FlexItem>
                    <FlexItem>
                        <strong>Important</strong>: If you are committing the saved custom resource
                        to a source control repository, replace the policy name in the{' '}
                        <code className="pf-v5-u-font-family-monospace">policyName</code> field to
                        avoid overwriting existing policies.
                    </FlexItem>
                </Flex>
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
