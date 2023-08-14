import React, { useState, useReducer } from 'react';
import PropTypes from 'prop-types';
import useDeepCompareEffect from 'use-deep-compare-effect';
import get from 'lodash/get';
import {
    Button,
    Dropdown,
    DropdownItem,
    DropdownPosition,
    DropdownToggle,
} from '@patternfly/react-core';

import CheckboxTable from 'Components/CheckboxTable';
import CloseButton from 'Components/CloseButton';
import Dialog from 'Components/Dialog';
import { DEFAULT_PAGE_SIZE } from 'Components/Table';
import TableHeader from 'Components/TableHeader';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd } from 'Components/Panel';
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
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';
import { filterAllowedSearch, convertToRestSearch, getHasSearchApplied } from 'utils/searchUtils';
import { getVersionedDocs } from 'utils/versioning';

import AutoUpgradeToggle from './Components/AutoUpgradeToggle';
import { clusterTablePollingInterval, getUpgradeableClusters } from './cluster.helpers';
import { getColumnsForClusters } from './clustersTableColumnDescriptors';
import AddClusterPrompt from './AddClusterPrompt';

function ClustersTablePanel({ selectedClusterId, setSelectedClusterId, searchOptions }) {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForAdministration = hasReadWriteAccess('Administration');
    const hasWriteAccessForCluster = hasReadWriteAccess('Cluster');
    const [isInstallMenuOpen, setIsInstallMenuOpen] = useState(false);

    function onToggleInstallMenu(newIsInstallMenuOpen) {
        setIsInstallMenuOpen(newIsInstallMenuOpen);
    }

    function onFocusInstallMenu() {
        const element = document.getElementById('toggle-descriptions');
        element.focus();
    }

    function onSelectInstallMenuItem() {
        setIsInstallMenuOpen(false);
        onFocusInstallMenu();
    }

    const metadata = useMetadata();

    const { searchFilter: pageSearch } = useURLSearch();

    const [checkedClusterIds, setCheckedClusters] = useState([]);
    const [upgradableClusters, setUpgradableClusters] = useState([]);
    const [pollingCount, setPollingCount] = useState(0);
    const [tableRef, setTableRef] = useState(null);
    const [showDialog, setShowDialog] = useState(false);
    const [fetchingClusters, setFetchingClusters] = useState(false);

    // Handle changes to applied search options.
    const [isViewFiltered, setIsViewFiltered] = useState(false);

    const [currentClusters, setCurrentClusters] = useState([]);
    const [clusterIdToRetentionInfo, setClusterIdToRetentionInfo] = useState({});

    function notificationsReducer(state, action) {
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

    const installMenuOptions = [
        version ? (
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
        <DropdownItem key="add" onClick={onAddCluster}>
            New cluster
        </DropdownItem>,
    ];

    function refreshClusterList(restSearch) {
        setFetchingClusters(true);
        return fetchClustersWithRetentionInfo(restSearch)
            .then((clustersResponse) => {
                setCurrentClusters(clustersResponse.clusters);
                setClusterIdToRetentionInfo(clustersResponse.clusterIdToRetentionInfo);
                setFetchingClusters(false);
            })
            .catch(() => {
                setFetchingClusters(false);
            });
    }

    const filteredSearch = filterAllowedSearch(searchOptions, pageSearch || {});
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

    function onAddCluster() {
        setSelectedClusterId('new');
    }

    function upgradeSingleCluster(id) {
        upgradeCluster(id)
            .then(() => {
                refreshClusterList();
            })
            .catch((error) => {
                const serverError = get(
                    error,
                    'response.data.message',
                    'An unknown error has occurred.'
                );
                const givenCluster = currentClusters.find((cluster) => cluster.id === id);
                const clusterName = givenCluster ? givenCluster.name : '-';
                const payload = `Failed to trigger upgrade for cluster ${clusterName}. Error: ${serverError}`;

                dispatch({ type: 'ADD_NOTIFICATION', payload });
            });
    }

    function upgradeSelectedClusters() {
        upgradeClusters(checkedClusterIds).then(() => {
            setCheckedClusters([]);

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
                setCheckedClusters([]);

                fetchClustersWithRetentionInfo().then((clustersResponse) => {
                    setCurrentClusters(clustersResponse.clusters);
                    setClusterIdToRetentionInfo(clustersResponse.clusterIdToRetentionInfo);
                });
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
                                Install cluster
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

    const onDeleteHandler = (cluster) => (e) => {
        e.stopPropagation();
        setCheckedClusters([cluster.id]);
        setShowDialog(true);
    };

    function toggleCluster(id) {
        const selection = toggleRow(id, checkedClusterIds);
        setCheckedClusters(selection);

        calculateUpgradeableClusters(selection);
    }

    function toggleAllClusters() {
        const rowsLength = checkedClusterIds.length;
        const ref = tableRef.reactTable;
        const selection = toggleSelectAll(rowsLength, checkedClusterIds, ref);
        setCheckedClusters(selection);

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

    const hasSearchApplied = getHasSearchApplied(filteredSearch);

    return (
        <div className="overflow-hidden w-full">
            <PanelNew testid="panel">
                <PanelHead>
                    {headerComponent}
                    <PanelHeadEnd>{headerActions}</PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    {messages.length > 0 && (
                        <div className="flex flex-col w-full items-center bg-warning-200 text-warning-8000 justify-center font-700 text-center">
                            {messages}
                        </div>
                    )}
                    {(!fetchingClusters || pollingCount > 0) &&
                        currentClusters.length <= 0 &&
                        !hasSearchApplied && <AddClusterPrompt />}
                    {(!fetchingClusters || pollingCount > 0) &&
                        (currentClusters.length > 0 || hasSearchApplied) && (
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
                        )}
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
        </div>
    );
}

ClustersTablePanel.propTypes = {
    selectedClusterId: PropTypes.string,
    setSelectedClusterId: PropTypes.func.isRequired,
    searchOptions: PropTypes.arrayOf(PropTypes.string),
};

ClustersTablePanel.defaultProps = {
    selectedClusterId: null,
    searchOptions: [],
};

export default ClustersTablePanel;
