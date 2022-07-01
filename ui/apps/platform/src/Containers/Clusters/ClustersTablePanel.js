import React, { useState, useReducer } from 'react';
import PropTypes from 'prop-types';
import useDeepCompareEffect from 'use-deep-compare-effect';
import { DownloadCloud, Plus, Trash2 } from 'react-feather';
import get from 'lodash/get';

import CheckboxTable from 'Components/CheckboxTable';
import CloseButton from 'Components/CloseButton';
import Dialog from 'Components/Dialog';
import PanelButton from 'Components/PanelButton';
import { DEFAULT_PAGE_SIZE } from 'Components/Table';
import TableHeader from 'Components/TableHeader';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd } from 'Components/Panel';
import useInterval from 'hooks/useInterval';
import useMetadata from 'hooks/useMetadata';
import useURLSearch from 'hooks/useURLSearch';
import {
    fetchClustersWithRetentionInfo,
    deleteClusters,
    upgradeClusters,
    upgradeCluster,
} from 'services/ClustersService';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';
import { filterAllowedSearch, convertToRestSearch } from 'utils/searchUtils';

import AutoUpgradeToggle from './Components/AutoUpgradeToggle';
import { clusterTablePollingInterval, getUpgradeableClusters } from './cluster.helpers';
import { getColumnsForClusters } from './clustersTableColumnDescriptors';

function ClustersTablePanel({ selectedClusterId, setSelectedClusterId, searchOptions }) {
    const metadata = useMetadata();

    const { searchFilter: pageSearch } = useURLSearch();

    const [checkedClusterIds, setCheckedClusters] = useState([]);
    const [upgradableClusters, setUpgradableClusters] = useState([]);
    const [pollingCount, setPollingCount] = useState(0);
    const [tableRef, setTableRef] = useState(null);
    const [showDialog, setShowDialog] = useState(false);

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

    function refreshClusterList(restSearch) {
        return fetchClustersWithRetentionInfo(restSearch).then((clustersResponse) => {
            setCurrentClusters(clustersResponse.clusters);
            setClusterIdToRetentionInfo(clustersResponse.clusterIdToRetentionInfo);
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
            type="Cluster"
            isViewFiltered={isViewFiltered}
        />
    );

    const headerActions = (
        <>
            <AutoUpgradeToggle />
            <PanelButton
                icon={<DownloadCloud className="h-4 w-4 ml-1" />}
                tooltip={`Upgrade (${upgradableClusters.length})`}
                className="btn btn-tertiary ml-2"
                onClick={upgradeSelectedClusters}
                disabled={upgradableClusters.length === 0 || !!selectedClusterId}
            >
                {`Upgrade (${upgradableClusters.length})`}
            </PanelButton>
            <PanelButton
                icon={<Trash2 className="h-4 w-4 ml-1" />}
                tooltip={`Delete (${checkedClusterIds.length})`}
                className="btn btn-alert ml-2"
                onClick={deleteSelectedClusters}
                disabled={checkedClusterIds.length === 0 || !!selectedClusterId}
            >
                {`Delete (${checkedClusterIds.length})`}
            </PanelButton>
            <PanelButton
                icon={<Plus className="h-4 w-4 ml-1" />}
                tooltip="New Cluster"
                className="btn btn-base ml-2 mr-4"
                onClick={onAddCluster}
                disabled={!!selectedClusterId}
            >
                New Cluster
            </PanelButton>
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
