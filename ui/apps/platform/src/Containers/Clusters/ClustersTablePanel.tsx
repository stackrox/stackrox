import React, { useCallback, useEffect, useMemo, useReducer, useState } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import {
    Alert,
    Bullseye,
    Button,
    DropdownItem,
    PageSection,
    Spinner,
    Text,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import MenuDropdown from 'Components/PatternFly/MenuDropdown';
import CloseButton from 'Components/CloseButton';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import {
    makeFilterChipDescriptors,
    onURLSearch,
} from 'Components/CompoundSearchFilter/utils/utils';
import Dialog from 'Components/Dialog';
import LinkShim from 'Components/PatternFly/LinkShim';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import SearchFilterInput from 'Components/SearchFilterInput';
import useAnalytics, {
    LEGACY_SECURE_A_CLUSTER_LINK_CLICKED,
    SECURE_A_CLUSTER_LINK_CLICKED,
    CRS_SECURE_A_CLUSTER_LINK_CLICKED,
} from 'hooks/useAnalytics';
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
import type { SearchCategory } from 'services/SearchService';
import type { Cluster } from 'types/cluster.proto';
import type { ClusterIdToRetentionInfo } from 'types/clusterService.proto';
import { toggleRow } from 'utils/checkboxUtils';
import { getTableUIState } from 'utils/getTableUIState';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { convertToRestSearch, getHasSearchApplied } from 'utils/searchUtils';
import {
    clustersBasePath,
    clustersDelegatedScanningPath,
    clustersDiscoveredClustersPath,
    clustersInitBundlesPath,
    clustersSecureClusterPath,
    clustersSecureClusterCrsPath,
    clustersClusterRegistrationSecretsPath,
} from 'routePaths';

import ClustersTable from './ClustersTable';
import AutoUpgradeToggle from './Components/AutoUpgradeToggle';
import SecureClusterModal from './InitBundles/SecureClusterModal';
import { clusterTablePollingInterval, getUpgradeableClusters } from './cluster.helpers';
import NoClustersPage from './NoClustersPage';
import { searchFilterConfig } from './searchFilterConfig';

const filterChipGroupDescriptors = makeFilterChipDescriptors(searchFilterConfig);

export type ClustersTablePanelProps = {
    selectedClusterId: string;
    searchOptions: SearchCategory[];
};

function ClustersTablePanel({ selectedClusterId, searchOptions }: ClustersTablePanelProps) {
    const { analyticsTrack } = useAnalytics();
    const navigate = useNavigate();

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isClustersPageMigrationEnabled = isFeatureFlagEnabled('ROX_CLUSTERS_PAGE_MIGRATION_UI');

    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasReadAccessForAdministration = hasReadAccess('Administration');
    const hasWriteAccessForAdministration = hasReadWriteAccess('Administration');
    const hasWriteAccessForCluster = hasReadWriteAccess('Cluster');

    const { currentUser } = useAuthStatus();
    const hasAdminRole = Boolean(currentUser?.userInfo?.roles.some(({ name }) => name === 'Admin')); // optional chaining just in case of the unexpected

    const [isModalOpen, setIsModalOpen] = useState(false);

    function onFocusInstallMenu() {
        const element = document.getElementById('toggle-descriptions');
        if (element !== null) {
            element.focus();
        }
    }

    function onSelectInstallMenuItem() {
        onFocusInstallMenu();
    }

    const metadata = useMetadata();

    const { searchFilter, setSearchFilter } = useURLSearch();

    const [checkedClusterIds, setCheckedClusterIds] = useState<string[]>([]);
    const [upgradableClusters, setUpgradableClusters] = useState<Cluster[]>([]);
    const [showDialog, setShowDialog] = useState(false);
    const [errorMessage, setErrorMessage] = useState('');
    const [hasFetchedClusters, setHasFetchedClusters] = useState(false);
    const [isLoadingVisible, setIsLoadingVisible] = useState(false);

    const restSearch = useMemo(() => convertToRestSearch(searchFilter ?? {}), [searchFilter]);

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

    const fetchClustersList = useCallback(
        (showLoadingSpinner: boolean) => {
            if (showLoadingSpinner) {
                setIsLoadingVisible(true);
            }

            fetchClustersWithRetentionInfo(restSearch)
                .then(({ clusters, clusterIdToRetentionInfo }) => {
                    setCurrentClusters(clusters);
                    setClusterIdToRetentionInfo(clusterIdToRetentionInfo);
                    setErrorMessage('');
                    setHasFetchedClusters(true);
                })
                .catch((err) => setErrorMessage(getAxiosErrorMessage(err)))
                .finally(() => showLoadingSpinner && setIsLoadingVisible(false));
        },
        [restSearch]
    );

    useEffect(() => {
        fetchClustersList(true);
    }, [fetchClustersList]);

    useInterval(() => fetchClustersList(false), clusterTablePollingInterval);

    const tableState = getTableUIState({
        isLoading: !hasFetchedClusters || isLoadingVisible,
        data: currentClusters,
        error: errorMessage ? new Error(errorMessage) : undefined,
        searchFilter,
    });

    // Before there is a response:
    // TODO: can be deleted once the ROX_CLUSTERS_PAGE_MIGRATION_UI flag is removed
    if (!hasFetchedClusters && !isClustersPageMigrationEnabled) {
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
                        <Spinner />
                    )}
                </Bullseye>
            </PageSection>
        );
    }

    const hasSearchApplied = getHasSearchApplied(searchFilter);

    // PatternFly clusters page: reconsider whether to factor out minimal common heading.
    //
    // After there is a response, if there are no clusters nor search filter:
    // TODO: can be deleted once the ROX_CLUSTERS_PAGE_MIGRATION_UI flag is removed
    if (currentClusters.length === 0 && !hasSearchApplied && !isClustersPageMigrationEnabled) {
        return <NoClustersPage isModalOpen={isModalOpen} setIsModalOpen={setIsModalOpen} />;
    }

    function upgradeSingleCluster(id) {
        upgradeCluster(id)
            .then(() => {
                fetchClustersList(true);
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

            fetchClustersList(true);
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
        // TODO uncouple from CheckboxTable?
        const selection = toggleRow(id, checkedClusterIds);
        setCheckedClusterIds(selection);

        calculateUpgradeableClusters(selection);
    }

    function toggleAllClusters() {
        // TODO uncouple from CheckboxTable?
        /*
        const rowsLength = checkedClusterIds.length;
        const ref = tableRef?.reactTable;
        const selection = toggleSelectAll(rowsLength, checkedClusterIds, ref);
        setCheckedClusterIds(selection);

        calculateUpgradeableClusters(selection);
        */
    }

    // After there is a response, if there are clusters or search filter.
    // Conditionally render a subsequent error in addition to most recent successful respnse.
    return (
        <>
            <PageSection variant="light" component="div">
                <Toolbar inset={{ default: 'insetNone' }} className="pf-v5-u-pb-0">
                    <ToolbarContent>
                        <Title headingLevel="h1">Clusters</Title>
                        <ToolbarGroup
                            className="pf-v5-u-flex-wrap"
                            variant="button-group"
                            align={{ default: 'alignRight' }}
                        >
                            {hasReadAccessForAdministration && (
                                <ToolbarItem>
                                    <Button
                                        variant="secondary"
                                        component={LinkShim}
                                        href={clustersDelegatedScanningPath}
                                    >
                                        Delegated image scanning
                                    </Button>
                                </ToolbarItem>
                            )}
                            {hasReadAccessForAdministration && (
                                <ToolbarItem>
                                    <Button
                                        variant="secondary"
                                        component={LinkShim}
                                        href={clustersDiscoveredClustersPath}
                                    >
                                        Discovered clusters
                                    </Button>
                                </ToolbarItem>
                            )}
                            {hasAdminRole && (
                                <ToolbarItem>
                                    <Button
                                        variant="secondary"
                                        component={LinkShim}
                                        href={clustersInitBundlesPath}
                                    >
                                        Init bundles
                                    </Button>
                                </ToolbarItem>
                            )}
                            {hasAdminRole && (
                                <ToolbarItem>
                                    <Button
                                        variant="secondary"
                                        component={LinkShim}
                                        href={clustersClusterRegistrationSecretsPath}
                                    >
                                        Cluster registration secrets
                                    </Button>
                                </ToolbarItem>
                            )}
                            {hasWriteAccessForCluster && (
                                <ToolbarItem>
                                    <MenuDropdown
                                        toggleText="Secure a cluster"
                                        onSelect={onSelectInstallMenuItem}
                                        popperProps={{
                                            position: 'end',
                                        }}
                                    >
                                        <DropdownItem
                                            key="init-bundle"
                                            onClick={() => {
                                                analyticsTrack({
                                                    event: SECURE_A_CLUSTER_LINK_CLICKED,
                                                    properties: {
                                                        source: 'Secure a Cluster Dropdown',
                                                    },
                                                });
                                                navigate(clustersSecureClusterPath);
                                            }}
                                        >
                                            Init bundle installation methods
                                        </DropdownItem>
                                        <DropdownItem
                                            key="cluster-registration-secret"
                                            onClick={() => {
                                                analyticsTrack({
                                                    event: CRS_SECURE_A_CLUSTER_LINK_CLICKED,
                                                    properties: {
                                                        source: 'Secure a Cluster Dropdown',
                                                    },
                                                });
                                                navigate(clustersSecureClusterCrsPath);
                                            }}
                                        >
                                            Cluster registration secret installation methods
                                        </DropdownItem>
                                        <DropdownItem
                                            key="legacy"
                                            onClick={() => {
                                                analyticsTrack({
                                                    event: LEGACY_SECURE_A_CLUSTER_LINK_CLICKED,
                                                    properties: {
                                                        source: 'Secure a Cluster Dropdown',
                                                    },
                                                });
                                                navigate(`${clustersBasePath}/new`);
                                            }}
                                        >
                                            Legacy installation method
                                        </DropdownItem>
                                    </MenuDropdown>
                                </ToolbarItem>
                            )}
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
                <Text className="pf-v5-u-font-size-md">
                    View the status of secured cluster services
                </Text>
            </PageSection>
            <PageSection>
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarGroup
                            variant="filter-group"
                            className="pf-v5-u-flex-grow-1 pf-v5-u-flex-shrink-1"
                        >
                            <ToolbarItem variant="search-filter" className="pf-v5-u-w-100">
                                {isClustersPageMigrationEnabled ? (
                                    <CompoundSearchFilter
                                        config={searchFilterConfig}
                                        searchFilter={searchFilter}
                                        onSearch={(payload) =>
                                            onURLSearch(searchFilter, setSearchFilter, payload)
                                        }
                                    />
                                ) : (
                                    <SearchFilterInput
                                        className="w-full"
                                        searchFilter={searchFilter}
                                        searchOptions={searchOptions}
                                        searchCategory="CLUSTERS"
                                        placeholder="Filter clusters"
                                        handleChangeSearchFilter={setSearchFilter}
                                    />
                                )}
                            </ToolbarItem>
                        </ToolbarGroup>
                        <ToolbarGroup variant="button-group" align={{ default: 'alignRight' }}>
                            {hasWriteAccessForAdministration && (
                                <ToolbarItem className="pf-v5-u-align-self-center">
                                    <AutoUpgradeToggle />
                                </ToolbarItem>
                            )}
                            {hasWriteAccessForAdministration && (
                                <ToolbarItem>
                                    <Button
                                        variant="secondary"
                                        onClick={upgradeSelectedClusters}
                                        isDisabled={
                                            upgradableClusters.length === 0 || !!selectedClusterId
                                        }
                                    >
                                        {`Upgrade (${upgradableClusters.length})`}
                                    </Button>
                                </ToolbarItem>
                            )}
                            {hasWriteAccessForCluster && (
                                <ToolbarItem>
                                    <Button
                                        variant="danger"
                                        onClick={deleteSelectedClusters}
                                        isDisabled={
                                            checkedClusterIds.length === 0 || !!selectedClusterId
                                        }
                                    >
                                        {`Delete (${checkedClusterIds.length})`}
                                    </Button>
                                </ToolbarItem>
                            )}
                        </ToolbarGroup>
                        {isClustersPageMigrationEnabled && (
                            <ToolbarGroup className="pf-v5-u-w-100">
                                <SearchFilterChips
                                    searchFilter={searchFilter}
                                    onFilterChange={setSearchFilter}
                                    filterChipGroupDescriptors={filterChipGroupDescriptors}
                                />
                            </ToolbarGroup>
                        )}
                    </ToolbarContent>
                </Toolbar>
                {errorMessage && !isClustersPageMigrationEnabled && (
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
                <ClustersTable
                    centralVersion={metadata.version}
                    clusterIdToRetentionInfo={clusterIdToRetentionInfo}
                    tableState={tableState}
                    selectedClusterIds={checkedClusterIds}
                    onClearFilters={() => setSearchFilter({})}
                    onDeleteCluster={onDeleteHandler}
                    toggleAllClusters={toggleAllClusters}
                    toggleCluster={toggleCluster}
                    upgradeSingleCluster={upgradeSingleCluster}
                />
            </PageSection>
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
        </>
    );
}

export default ClustersTablePanel;
