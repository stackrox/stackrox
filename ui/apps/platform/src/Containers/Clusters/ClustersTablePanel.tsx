import React, { ReactElement, useState, useReducer } from 'react';
import { Link, useHistory } from 'react-router-dom';
import useDeepCompareEffect from 'use-deep-compare-effect';
import {
    Alert,
    Bullseye,
    Button,
    Dropdown,
    DropdownItem,
    DropdownPosition,
    DropdownToggle,
    PageSection,
    Spinner,
} from '@patternfly/react-core';

import CheckboxTable from 'Components/CheckboxTable';
import CloseButton from 'Components/CloseButton';
import Dialog from 'Components/Dialog';
import PageHeader from 'Components/PageHeader';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd } from 'Components/Panel';
import LinkShim from 'Components/PatternFly/LinkShim';
import SearchFilterInput from 'Components/SearchFilterInput';
import { DEFAULT_PAGE_SIZE } from 'Components/Table';
import TableHeader from 'Components/TableHeader';
import useAuthStatus from 'hooks/useAuthStatus';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useInterval from 'hooks/useInterval';
import useMetadata from 'hooks/useMetadata';
import usePermissions from 'hooks/usePermissions';
import useURLSearch from 'hooks/useURLSearch';
import {
    fetchClustersWithRetentionInfo,
    deleteClusters,
    upgradeClusters,
    upgradeCluster,
} from 'services/ClustersService';
import { SearchCategory } from 'services/SearchService';
import { RestSearchOption } from 'services/searchOptionsToQuery';
import { Cluster } from 'types/cluster.proto';
import { ClusterIdToRetentionInfo } from 'types/clusterService.proto';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { filterAllowedSearch, convertToRestSearch, getHasSearchApplied } from 'utils/searchUtils';
import { getVersionedDocs } from 'utils/versioning';
import {
    clustersBasePath,
    clustersDelegatedScanningPath,
    clustersSecureClusterPath,
} from 'routePaths';

import AutoUpgradeToggle from './Components/AutoUpgradeToggle';
import ManageTokensButton from './Components/ManageTokensButton';
import SecureClusterModal from './InitBundles/SecureClusterModal';
import { clusterTablePollingInterval, getUpgradeableClusters } from './cluster.helpers';
import { getColumnsForClusters } from './clustersTableColumnDescriptors';
import AddClusterPrompt from './AddClusterPrompt';
import NoClustersPage from './NoClustersPage';

export type ClustersTablePanelProps = {
    selectedClusterId: string;
    searchOptions: SearchCategory[];
};

function ClustersTablePanel({
    selectedClusterId,
    searchOptions,
}: ClustersTablePanelProps): ReactElement {
    const history = useHistory();

    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasReadAccessForDelegatedScanning = hasReadAccess('Administration');
    const hasWriteAccessForAdministration = hasReadWriteAccess('Administration');
    const hasWriteAccessForCluster = hasReadWriteAccess('Cluster');

    const { currentUser } = useAuthStatus();
    const hasAdminRole = Boolean(currentUser?.userInfo?.roles.some(({ name }) => name === 'Admin')); // optional chaining just in case of the unexpected

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isMoveInitBundlesEnabled = isFeatureFlagEnabled('ROX_MOVE_INIT_BUNDLES_UI');

    const [isInstallMenuOpen, setIsInstallMenuOpen] = useState(false);
    const [isModalOpen, setIsModalOpen] = useState(false);

    function onToggleInstallMenu(newIsInstallMenuOpen) {
        setIsInstallMenuOpen(newIsInstallMenuOpen);
    }

    function onFocusInstallMenu() {
        const element = document.getElementById('toggle-descriptions');
        if (element !== null) {
            element.focus();
        }
    }

    function onSelectInstallMenuItem() {
        setIsInstallMenuOpen(false);
        onFocusInstallMenu();
    }

    const metadata = useMetadata();

    const { searchFilter, setSearchFilter } = useURLSearch();

    const [checkedClusterIds, setCheckedClusterIds] = useState<string[]>([]);
    const [upgradableClusters, setUpgradableClusters] = useState<Cluster[]>([]);
    const [pollingCount, setPollingCount] = useState(0);
    const [tableRef, setTableRef] = useState<CheckboxTable | null>(null);
    const [showDialog, setShowDialog] = useState(false);
    const [errorMessage, setErrorMessage] = useState('');
    const [hasFetchedClusters, setHasFetchedClusters] = useState(false);

    // Handle changes to applied search options.
    const [isViewFiltered, setIsViewFiltered] = useState(false);

    const [currentClusters, setCurrentClusters] = useState<Cluster[]>([]);
    const [clusterIdToRetentionInfo, setClusterIdToRetentionInfo] =
        useState<ClusterIdToRetentionInfo>({});

    type NotificationAction = {
        type: 'ADD_NOTIFICATION' | 'REMOVE_NOTIFICATION';
        payload: string;
    };

    function notificationsReducer(state: string[], action: NotificationAction) {
        switch (action.type) {
            case 'ADD_NOTIFICATION': {
                return [...state, action.payload];
            }
            case 'REMOVE_NOTIFICATION': {
                return state.filter((note) => note !== action.payload);
            }
            default: {
                return state;
            }
        }
    }
    const [notifications, dispatch] = useReducer(notificationsReducer, []);

    const messages = notifications.map((note) => (
        <div
            key={note}
            className="flex flex-1 border-b border-base-400 items-center justify-end relative py-0 pl-3 w-full"
        >
            <span className="w-full">{note}</span>
            <CloseButton
                onClose={() => {
                    dispatch({ type: 'REMOVE_NOTIFICATION', payload: note });
                }}
                className="border-base-400 border-l"
            />
        </div>
    ));

    const { version } = metadata;

    const pageHeader = (
        <PageHeader header="Clusters" subHeader="Resource list">
            <div className="flex flex-1 items-center justify-end">
                <SearchFilterInput
                    className="w-full"
                    searchFilter={searchFilter}
                    searchOptions={searchOptions}
                    searchCategory="CLUSTERS"
                    placeholder="Filter clusters"
                    handleChangeSearchFilter={setSearchFilter}
                />
                {hasReadAccessForDelegatedScanning && (
                    <div className="flex items-center ml-4 mr-1">
                        <Button
                            variant="secondary"
                            component={LinkShim}
                            href={clustersDelegatedScanningPath}
                        >
                            Delegated scanning
                        </Button>
                    </div>
                )}
                {hasAdminRole && (
                    <div className="flex items-center ml-1">
                        <Button variant="tertiary">
                            <ManageTokensButton />
                        </Button>
                    </div>
                )}
            </div>
        </PageHeader>
    );

    /* eslint-disable no-nested-ternary */
    const installMenuOptions = [
        isMoveInitBundlesEnabled ? (
            <DropdownItem
                key="init-bundle"
                component={
                    <Link to={clustersSecureClusterPath}>Init bundle installation methods</Link>
                }
            />
        ) : version ? (
            <DropdownItem
                key="link"
                description="Cluster installation guides"
                href={getVersionedDocs(version, 'installing/acs-installation-platforms.html')}
                target="_blank"
                rel="noopener noreferrer"
            >
                View instructions
            </DropdownItem>
        ) : (
            <DropdownItem key="version-missing" isPlainText>
                Instructions unavailable; version missing
            </DropdownItem>
        ),
        <DropdownItem
            key="legacy"
            component={
                <Link to={`${clustersBasePath}/new`}>
                    {isMoveInitBundlesEnabled ? 'Legacy installation method' : 'New cluster'}
                </Link>
            }
        />,
    ];
    /* eslint-enable no-nested-ternary */

    function refreshClusterList(restSearch?: RestSearchOption[]) {
        // Although return works around typescript-eslint/no-floating-promises error elsewhere,
        // removed here because it caused the error for callers.
        // Anyway, catch block would be better.
        fetchClustersWithRetentionInfo(restSearch)
            .then((clustersResponse) => {
                setCurrentClusters(clustersResponse.clusters);
                setClusterIdToRetentionInfo(clustersResponse.clusterIdToRetentionInfo);
                setErrorMessage('');
                setHasFetchedClusters(true);
            })
            .catch((error) => {
                setErrorMessage(getAxiosErrorMessage(error));
            });
    }

    const filteredSearch = filterAllowedSearch(searchOptions, searchFilter || {});
    const restSearch = convertToRestSearch(filteredSearch || {});
    useDeepCompareEffect(() => {
        if (restSearch.length) {
            setIsViewFiltered(true);
        } else {
            setIsViewFiltered(false);
        }

        refreshClusterList(restSearch);
    }, [restSearch, pollingCount]);

    // use a custom hook to set up polling, thanks Dan Abramov and Rob Stark
    useInterval(() => {
        setPollingCount(pollingCount + 1);
    }, clusterTablePollingInterval);

    // Do not render page heading now because of current NoClustersPage design (rendered below).
    // PatternFly clusters page: reconsider whether to factor out minimal common heading.
    //
    // Before there is a response:
    if (!hasFetchedClusters) {
        return (
            <PageSection variant="light">
                <Bullseye>
                    {errorMessage ? (
                        <Alert
                            variant="warning"
                            isInline
                            title="Unable to fetch clusters"
                            component="p"
                        >
                            {errorMessage}
                        </Alert>
                    ) : (
                        <Spinner isSVG />
                    )}
                </Bullseye>
            </PageSection>
        );
    }

    const hasSearchApplied = getHasSearchApplied(filteredSearch);

    // PatternFly clusters page: reconsider whether to factor out minimal common heading.
    //
    // After there is a response, if there are no clusters nor search filter:
    if (currentClusters.length === 0 && !hasSearchApplied) {
        return isMoveInitBundlesEnabled ? (
            <NoClustersPage isModalOpen={isModalOpen} setIsModalOpen={setIsModalOpen} />
        ) : (
            <PageSection variant="light">
                <Bullseye>
                    <AddClusterPrompt />
                </Bullseye>
            </PageSection>
        );
    }

    function setSelectedClusterId(cluster: Cluster) {
        history.push(`${clustersBasePath}/${cluster.id}`);
    }

    function upgradeSingleCluster(id) {
        upgradeCluster(id)
            .then(() => {
                refreshClusterList();
            })
            .catch((error) => {
                const serverError = getAxiosErrorMessage(error);
                const givenCluster = currentClusters.find((cluster) => cluster.id === id);
                const clusterName = givenCluster ? givenCluster.name : '-';
                const payload = `Failed to trigger upgrade for cluster ${clusterName}. Error: ${serverError}`;

                dispatch({ type: 'ADD_NOTIFICATION', payload });
            });
    }

    function upgradeSelectedClusters() {
        // Although return works around typescript-eslint/no-floating-promises error,
        // catch block would be better.
        return upgradeClusters(checkedClusterIds).then(() => {
            setCheckedClusterIds([]);

            refreshClusterList();
        });
    }

    function deleteSelectedClusters() {
        setShowDialog(true);
    }

    function hideDialog() {
        setShowDialog(false);
    }

    function makeDeleteRequest() {
        deleteClusters(checkedClusterIds)
            .then(() => {
                setCheckedClusterIds([]);

                return fetchClustersWithRetentionInfo().then((clustersResponse) => {
                    setCurrentClusters(clustersResponse.clusters);
                    setClusterIdToRetentionInfo(clustersResponse.clusterIdToRetentionInfo);
                });
            })
            .catch(() => {
                // TODO render error in dialogand move finally code to then block.
            })
            .finally(() => {
                setShowDialog(false);
            });
    }

    const headerComponent = (
        <TableHeader
            length={currentClusters?.length || 0}
            type="cluster"
            isViewFiltered={isViewFiltered}
        />
    );

    const headerActions = (
        <>
            {hasWriteAccessForAdministration && (
                <>
                    <AutoUpgradeToggle />
                    <Button
                        variant="secondary"
                        className="pf-u-ml-sm"
                        onClick={upgradeSelectedClusters}
                        isDisabled={upgradableClusters.length === 0 || !!selectedClusterId}
                    >
                        {`Upgrade (${upgradableClusters.length})`}
                    </Button>
                </>
            )}
            {hasWriteAccessForCluster && (
                <>
                    <Button
                        variant="danger"
                        className="pf-u-ml-sm pf-u-mr-sm"
                        onClick={deleteSelectedClusters}
                        isDisabled={checkedClusterIds.length === 0 || !!selectedClusterId}
                    >
                        {`Delete (${checkedClusterIds.length})`}
                    </Button>
                    <Dropdown
                        className="mr-4"
                        onSelect={onSelectInstallMenuItem}
                        toggle={
                            <DropdownToggle
                                id="install-toggle"
                                toggleVariant="secondary"
                                onToggle={onToggleInstallMenu}
                            >
                                Secure a cluster
                            </DropdownToggle>
                        }
                        position={DropdownPosition.right}
                        isOpen={isInstallMenuOpen}
                        dropdownItems={installMenuOptions}
                    />
                </>
            )}
        </>
    );

    function calculateUpgradeableClusters(selection) {
        const currentlySelectedClusters = currentClusters.filter((cluster) =>
            selection.includes(cluster.id)
        );

        const upgradeableList = getUpgradeableClusters(currentlySelectedClusters);

        setUpgradableClusters(upgradeableList);
    }

    const onDeleteHandler = (cluster: Cluster) => (e) => {
        e.stopPropagation();
        setCheckedClusterIds([cluster.id]);
        setShowDialog(true);
    };

    function toggleCluster(id) {
        const selection = toggleRow(id, checkedClusterIds);
        setCheckedClusterIds(selection);

        calculateUpgradeableClusters(selection);
    }

    function toggleAllClusters() {
        const rowsLength = checkedClusterIds.length;
        const ref = tableRef?.reactTable;
        const selection = toggleSelectAll(rowsLength, checkedClusterIds, ref);
        setCheckedClusterIds(selection);

        calculateUpgradeableClusters(selection);
    }

    const columnOptions = {
        clusterIdToRetentionInfo,
        hasWriteAccessForCluster,
        metadata,
        rowActions: {
            onDeleteHandler,
            upgradeSingleCluster,
        },
    };
    const clusterColumns = getColumnsForClusters(columnOptions);

    // Because clusters are not paginated, make the list display them all.
    const pageSize =
        currentClusters.length <= DEFAULT_PAGE_SIZE ? DEFAULT_PAGE_SIZE : currentClusters.length;

    // After there is a response, if there are clusters or search filter.
    // Conditionally render a subsequent error in addition to most recent successful respnse.
    return (
        <div className="overflow-hidden w-full">
            {pageHeader}
            <PanelNew testid="panel">
                <PanelHead>
                    {headerComponent}
                    <PanelHeadEnd>{headerActions}</PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    {errorMessage && (
                        <Alert
                            variant="warning"
                            isInline
                            title="Unable to fetch clusters"
                            component="p"
                        >
                            {errorMessage}
                        </Alert>
                    )}
                    {messages.length > 0 && (
                        <div className="flex flex-col w-full items-center bg-warning-200 text-warning-8000 justify-center font-700 text-center">
                            {messages}
                        </div>
                    )}
                    <div data-testid="clusters-table" className="h-full w-full">
                        <CheckboxTable
                            ref={(table) => {
                                setTableRef(table);
                            }}
                            rows={currentClusters}
                            columns={clusterColumns}
                            onRowClick={setSelectedClusterId}
                            toggleRow={toggleCluster}
                            toggleSelectAll={toggleAllClusters}
                            selection={checkedClusterIds}
                            selectedRowId={selectedClusterId}
                            noDataText="No clusters to show."
                            minRows={20}
                            pageSize={pageSize}
                        />
                    </div>
                </PanelBody>
            </PanelNew>
            <Dialog
                className="w-1/3"
                isOpen={showDialog}
                text={`Deleting a cluster configuration doesn't remove security services running in the cluster. To remove them, run the "delete-sensor.sh" script from the sensor installation bundle. Are you sure you want to delete ${checkedClusterIds.length} cluster(s)?`}
                onConfirm={makeDeleteRequest}
                confirmText="Delete"
                onCancel={hideDialog}
                isDestructive
            />
            <SecureClusterModal isModalOpen={isModalOpen} setIsModalOpen={setIsModalOpen} />
        </div>
    );
}

export default ClustersTablePanel;
