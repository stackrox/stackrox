import { useCallback, useState } from 'react';
import {
    Alert,
    Button,
    Flex,
    FlexItem,
    Modal,
    ModalBody,
    ModalFooter,
    ModalHeader,
    PageSection,
    Pagination,
    TextInput,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { LockIcon, LockOpenIcon } from '@patternfly/react-icons';

import PageTitle from 'Components/PageTitle';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useURLPagination from 'hooks/useURLPagination';
import useRestQuery from 'hooks/useRestQuery';
import useRestMutation from 'hooks/useRestMutation';
import { getTableUIState } from 'utils/getTableUIState';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import type { ProcessBaseline, ProcessBaselineKey } from 'types/processBaseline.proto';
import {
    fetchProcessBaselinesBulk,
    lockUnlockProcessBaselines,
    addProcessesToBaseline,
    removeProcessesFromBaseline,
} from 'services/ProcessBaselineService';
import type { ProcessBaselineQuery } from 'services/ProcessBaselineService';

const DEFAULT_PAGE_SIZE = 20;

type BulkAction = 'lock' | 'unlock' | 'addProcess' | 'removeProcess' | null;

function buildQuery(searchTerms: Record<string, string>): ProcessBaselineQuery {
    const query: ProcessBaselineQuery = {};
    if (searchTerms.cluster) {
        query.clusterIds = [searchTerms.cluster];
    }
    if (searchTerms.namespace) {
        query.namespaces = [searchTerms.namespace];
    }
    if (searchTerms.deploymentName) {
        query.deploymentNames = [searchTerms.deploymentName];
    }
    if (searchTerms.image) {
        query.images = [searchTerms.image];
    }
    if (searchTerms.containerName) {
        query.containerNames = [searchTerms.containerName];
    }
    return query;
}

function isLocked(baseline: ProcessBaseline): boolean {
    return Boolean(baseline.userLockedTimestamp || baseline.stackRoxLockedTimestamp);
}

function ProcessBaselinesPage() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_PAGE_SIZE);

    const [searchTerms, setSearchTerms] = useState<Record<string, string>>({});
    const [clusterInput, setClusterInput] = useState('');
    const [namespaceInput, setNamespaceInput] = useState('');
    const [deploymentNameInput, setDeploymentNameInput] = useState('');
    const [containerNameInput, setContainerNameInput] = useState('');

    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
    const [bulkAction, setBulkAction] = useState<BulkAction>(null);
    const [processNameInput, setProcessNameInput] = useState('');
    const [actionError, setActionError] = useState<string | null>(null);
    const [actionSuccess, setActionSuccess] = useState<string | null>(null);

    const query = buildQuery(searchTerms);

    const requestFn = useCallback(
        () => fetchProcessBaselinesBulk(query, page, perPage),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [JSON.stringify(query), page, perPage]
    );

    const { data, isLoading, error, refetch } = useRestQuery(requestFn);

    const baselines = data?.baselines ?? [];
    const totalCount = data?.totalCount ?? 0;

    const lockMutation = useRestMutation(lockUnlockProcessBaselines);
    const addMutation = useRestMutation(addProcessesToBaseline);
    const removeMutation = useRestMutation(removeProcessesFromBaseline);

    const tableState = getTableUIState({
        isLoading,
        data: baselines,
        error,
        searchFilter: searchTerms,
    });

    function getSelectedKeys(): ProcessBaselineKey[] {
        return baselines
            .filter((b) => selectedIds.has(b.id))
            .map((b) => b.key);
    }

    function handleSearch() {
        const terms: Record<string, string> = {};
        if (clusterInput.trim()) {
            terms.cluster = clusterInput.trim();
        }
        if (namespaceInput.trim()) {
            terms.namespace = namespaceInput.trim();
        }
        if (deploymentNameInput.trim()) {
            terms.deploymentName = deploymentNameInput.trim();
        }
        if (containerNameInput.trim()) {
            terms.containerName = containerNameInput.trim();
        }
        setSearchTerms(terms);
        setPage(1);
        setSelectedIds(new Set());
    }

    function handleClearSearch() {
        setClusterInput('');
        setNamespaceInput('');
        setDeploymentNameInput('');
        setContainerNameInput('');
        setSearchTerms({});
        setPage(1);
        setSelectedIds(new Set());
    }

    function toggleSelectAll() {
        if (selectedIds.size === baselines.length) {
            setSelectedIds(new Set());
        } else {
            setSelectedIds(new Set(baselines.map((b) => b.id)));
        }
    }

    function toggleSelect(id: string) {
        setSelectedIds((prev) => {
            const next = new Set(prev);
            if (next.has(id)) {
                next.delete(id);
            } else {
                next.add(id);
            }
            return next;
        });
    }

    function handleBulkAction() {
        const keys = getSelectedKeys();
        if (keys.length === 0) {
            return;
        }

        setActionError(null);
        setActionSuccess(null);

        const onSuccess = (message: string) => () => {
            setActionSuccess(message);
            setBulkAction(null);
            setProcessNameInput('');
            setSelectedIds(new Set());
            refetch();
        };
        const onError = (err: unknown) => {
            setActionError(getAxiosErrorMessage(err));
        };

        switch (bulkAction) {
            case 'lock':
                lockMutation.mutate({ keys, locked: true }, { onSuccess: onSuccess(`Locked ${keys.length} baseline(s)`), onError });
                break;
            case 'unlock':
                lockMutation.mutate({ keys, locked: false }, { onSuccess: onSuccess(`Unlocked ${keys.length} baseline(s)`), onError });
                break;
            case 'addProcess':
                if (!processNameInput.trim()) {
                    return;
                }
                addMutation.mutate(
                    { keys, addElements: [{ processName: processNameInput.trim() }] },
                    { onSuccess: onSuccess(`Added process "${processNameInput.trim()}" to ${keys.length} baseline(s)`), onError }
                );
                break;
            case 'removeProcess':
                if (!processNameInput.trim()) {
                    return;
                }
                removeMutation.mutate(
                    { keys, removeElements: [{ processName: processNameInput.trim() }] },
                    { onSuccess: onSuccess(`Removed process "${processNameInput.trim()}" from ${keys.length} baseline(s)`), onError }
                );
                break;
            default:
                break;
        }
    }

    const isMutating = lockMutation.isLoading || addMutation.isLoading || removeMutation.isLoading;
    const needsProcessName = bulkAction === 'addProcess' || bulkAction === 'removeProcess';

    return (
        <>
            <PageTitle title="Process Baselines" />
            <PageSection>
                <Title headingLevel="h1">Process Baselines</Title>
            </PageSection>
            <PageSection>
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    {actionSuccess && (
                        <Alert
                            variant="success"
                            title={actionSuccess}
                            isInline
                            actionClose={
                                <Button variant="plain" onClick={() => setActionSuccess(null)}>
                                    Dismiss
                                </Button>
                            }
                        />
                    )}
                    {actionError && (
                        <Alert
                            variant="danger"
                            title={actionError}
                            isInline
                            actionClose={
                                <Button variant="plain" onClick={() => setActionError(null)}>
                                    Dismiss
                                </Button>
                            }
                        />
                    )}
                    <Flex spaceItems={{ default: 'spaceItemsSm' }} alignItems={{ default: 'alignItemsFlexEnd' }}>
                        <FlexItem>
                            <TextInput
                                aria-label="Cluster ID"
                                placeholder="Cluster ID"
                                value={clusterInput}
                                onChange={(_event, value) => setClusterInput(value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                            />
                        </FlexItem>
                        <FlexItem>
                            <TextInput
                                aria-label="Namespace"
                                placeholder="Namespace"
                                value={namespaceInput}
                                onChange={(_event, value) => setNamespaceInput(value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                            />
                        </FlexItem>
                        <FlexItem>
                            <TextInput
                                aria-label="Deployment name"
                                placeholder="Deployment name"
                                value={deploymentNameInput}
                                onChange={(_event, value) => setDeploymentNameInput(value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                            />
                        </FlexItem>
                        <FlexItem>
                            <TextInput
                                aria-label="Container name"
                                placeholder="Container name"
                                value={containerNameInput}
                                onChange={(_event, value) => setContainerNameInput(value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                            />
                        </FlexItem>
                        <FlexItem>
                            <Button variant="primary" onClick={handleSearch}>
                                Search
                            </Button>
                        </FlexItem>
                        <FlexItem>
                            <Button variant="link" onClick={handleClearSearch}>
                                Clear
                            </Button>
                        </FlexItem>
                    </Flex>
                    <Toolbar>
                        <ToolbarContent>
                            <ToolbarGroup>
                                <ToolbarItem>
                                    <Button
                                        variant="secondary"
                                        icon={<LockIcon />}
                                        isDisabled={selectedIds.size === 0}
                                        onClick={() => setBulkAction('lock')}
                                    >
                                        Lock
                                    </Button>
                                </ToolbarItem>
                                <ToolbarItem>
                                    <Button
                                        variant="secondary"
                                        icon={<LockOpenIcon />}
                                        isDisabled={selectedIds.size === 0}
                                        onClick={() => setBulkAction('unlock')}
                                    >
                                        Unlock
                                    </Button>
                                </ToolbarItem>
                                <ToolbarItem>
                                    <Button
                                        variant="secondary"
                                        isDisabled={selectedIds.size === 0}
                                        onClick={() => setBulkAction('addProcess')}
                                    >
                                        Add process
                                    </Button>
                                </ToolbarItem>
                                <ToolbarItem>
                                    <Button
                                        variant="secondary"
                                        isDisabled={selectedIds.size === 0}
                                        onClick={() => setBulkAction('removeProcess')}
                                    >
                                        Remove process
                                    </Button>
                                </ToolbarItem>
                            </ToolbarGroup>
                            <ToolbarItem align={{ default: 'alignEnd' }} variant="pagination">
                                <Pagination
                                    itemCount={totalCount}
                                    page={page}
                                    onSetPage={(_, newPage) => setPage(newPage)}
                                    perPage={perPage}
                                    onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                    <Table variant="compact">
                        <Thead noWrap>
                            <Tr>
                                <Th
                                    select={{
                                        onSelect: toggleSelectAll,
                                        isSelected: baselines.length > 0 && selectedIds.size === baselines.length,
                                    }}
                                />
                                <Th width={15}>Deployment</Th>
                                <Th width={10}>Container</Th>
                                <Th width={10}>Cluster</Th>
                                <Th width={10}>Namespace</Th>
                                <Th width={10}>Status</Th>
                                <Th>Processes</Th>
                            </Tr>
                        </Thead>
                        <TbodyUnified
                            tableState={tableState}
                            colSpan={7}
                            emptyProps={{ message: 'No process baselines found. Use the search fields above to find baselines.' }}
                            filteredEmptyProps={{ onClearFilters: handleClearSearch }}
                            renderer={({ data: rows }) =>
                                rows.map((baseline, rowIndex) => {
                                    const locked = isLocked(baseline);
                                    const processNames = baseline.elements
                                        .map((el) => el.element.processName)
                                        .filter(Boolean)
                                        .join(', ');

                                    return (
                                        <Tbody key={baseline.id}>
                                            <Tr>
                                                <Td
                                                    select={{
                                                        rowIndex,
                                                        isSelected: selectedIds.has(baseline.id),
                                                        onSelect: () => toggleSelect(baseline.id),
                                                    }}
                                                />
                                                <Td dataLabel="Deployment">{baseline.key.deploymentId}</Td>
                                                <Td dataLabel="Container">{baseline.key.containerName}</Td>
                                                <Td dataLabel="Cluster">{baseline.key.clusterId}</Td>
                                                <Td dataLabel="Namespace">{baseline.key.namespace}</Td>
                                                <Td dataLabel="Status">
                                                    {locked ? (
                                                        <><LockIcon /> Locked</>
                                                    ) : (
                                                        <><LockOpenIcon /> Unlocked</>
                                                    )}
                                                </Td>
                                                <Td dataLabel="Processes">{processNames || '-'}</Td>
                                            </Tr>
                                        </Tbody>
                                    );
                                })
                            }
                        />
                    </Table>
                </Flex>
            </PageSection>
            {bulkAction && (
                <Modal
                    isOpen
                    onClose={() => { setBulkAction(null); setProcessNameInput(''); }}
                    variant="small"
                >
                    <ModalHeader
                        title={
                            bulkAction === 'lock' ? 'Lock baselines' :
                            bulkAction === 'unlock' ? 'Unlock baselines' :
                            bulkAction === 'addProcess' ? 'Add process to baselines' :
                            'Remove process from baselines'
                        }
                    />
                    <ModalBody>
                        {bulkAction === 'lock' && (
                            <p>Lock {selectedIds.size} selected baseline(s)? Locked baselines will flag any process not in the baseline as anomalous.</p>
                        )}
                        {bulkAction === 'unlock' && (
                            <p>Unlock {selectedIds.size} selected baseline(s)? Unlocked baselines will continue to learn new processes.</p>
                        )}
                        {needsProcessName && (
                            <TextInput
                                aria-label="Process name"
                                placeholder="Enter process name"
                                value={processNameInput}
                                onChange={(_event, value) => setProcessNameInput(value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleBulkAction()}
                            />
                        )}
                    </ModalBody>
                    <ModalFooter>
                        <Button
                            variant="primary"
                            onClick={handleBulkAction}
                            isLoading={isMutating}
                            isDisabled={isMutating || (needsProcessName && !processNameInput.trim())}
                        >
                            Confirm
                        </Button>
                        <Button
                            variant="link"
                            onClick={() => { setBulkAction(null); setProcessNameInput(''); }}
                        >
                            Cancel
                        </Button>
                    </ModalFooter>
                </Modal>
            )}
        </>
    );
}

export default ProcessBaselinesPage;
