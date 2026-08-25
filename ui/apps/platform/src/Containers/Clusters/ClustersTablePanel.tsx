import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom-v5-compat';
import {
    Button,
    Content,
    DropdownItem,
    Flex,
    FlexItem,
    PageSection,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import MenuDropdown from 'Components/PatternFly/MenuDropdown';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import {
    getSearchFilterConfigWithFeatureFlagDependency,
    updateSearchFilter,
} from 'Components/CompoundSearchFilter/utils/utils';
import Dialog from 'Components/Dialog';
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
import { deleteClusters, fetchClustersWithRetentionInfo } from 'services/ClustersService';
import type { Cluster } from 'types/cluster.proto';
import type { ClusterIdToRetentionInfo } from 'types/clusterService.proto';
import { getTableUIState } from 'utils/getTableUIState';
import {
    applyRegexSearchModifiers,
    convertToRestSearch,
    getHasSearchApplied,
} from 'utils/searchUtils';
import {
    clustersBasePath,
    clustersClusterRegistrationSecretsPath,
    clustersDelegatedScanningPath,
    clustersDiscoveredClustersPath,
    clustersInitBundlesPath,
    clustersSecureClusterCrsPath,
    clustersSecureClusterPath,
} from 'routePaths';

import ClustersTable from './ClustersTable';
import SecureClusterModal from './ClusterRegistrationSecrets/SecureClusterModal';
import { clusterTablePollingInterval } from './cluster.helpers';
import NoClustersPage from './NoClustersPage';
import { searchFilterConfig } from './searchFilterConfig';

export type ClustersTablePanelProps = {
    selectedClusterId: string;
};

function ClustersTablePanel({ selectedClusterId }: ClustersTablePanelProps) {
    const { analyticsTrack } = useAnalytics();
    const navigate = useNavigate();
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasReadAccessForAdministration = hasReadAccess('Administration');
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

    const filteredSearchFilterConfig = useMemo(
        () =>
            getSearchFilterConfigWithFeatureFlagDependency(
                isFeatureFlagEnabled,
                searchFilterConfig
            ),

        [isFeatureFlagEnabled]
    );

    const [checkedClusterIds, setCheckedClusterIds] = useState<string[]>([]);
    const [showDialog, setShowDialog] = useState(false);
    const [errorForClustersWithRetentionInfo, setErrorForClustersWithRetentionInfo] = useState<
        Error | undefined
    >(undefined);
    const [hasFetchedClusters, setHasFetchedClusters] = useState(false);
    const [isLoadingVisible, setIsLoadingVisible] = useState(false);

    const restSearch = useMemo(
        () => convertToRestSearch(applyRegexSearchModifiers(searchFilter ?? {})),
        [searchFilter]
    );

    const [currentClusters, setCurrentClusters] = useState<Cluster[]>([]);
    const [clusterIdToRetentionInfo, setClusterIdToRetentionInfo] =
        useState<ClusterIdToRetentionInfo>({});

    const fetchClustersList = useCallback(
        (showLoadingSpinner: boolean) => {
            if (showLoadingSpinner) {
                setIsLoadingVisible(true);
            }

            fetchClustersWithRetentionInfo(restSearch)
                .then(({ clusters, clusterIdToRetentionInfo }) => {
                    setCurrentClusters(clusters);
                    setClusterIdToRetentionInfo(clusterIdToRetentionInfo);
                    setErrorForClustersWithRetentionInfo(undefined);
                    setHasFetchedClusters(true);
                })
                .catch((err) => setErrorForClustersWithRetentionInfo(err))
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
        error: errorForClustersWithRetentionInfo,
        searchFilter,
    });

    const hasSearchApplied = getHasSearchApplied(searchFilter);

    // Reconsider whether to factor out minimal common heading.
    //
    // After there is a response, if there are no clusters nor search filter:
    // Too bad, so sad: flicker because ClustersTable encapsulates spinner.
    if (currentClusters.length === 0 && !hasSearchApplied) {
        return <NoClustersPage isModalOpen={isModalOpen} setIsModalOpen={setIsModalOpen} />;
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

    const onDeleteHandler = (cluster: Cluster) => (e) => {
        e.stopPropagation();
        setCheckedClusterIds([cluster.id]);
        setShowDialog(true);
    };

    function toggleCluster(id) {
        const selection = checkedClusterIds.includes(id)
            ? checkedClusterIds.filter((checkedId) => checkedId !== id)
            : [...checkedClusterIds, id];

        setCheckedClusterIds(selection);
    }

    function toggleAllClusters() {
        // If all are selected in the entire table, all become unselected in the table.
        // If some or none are selected, all become selected on that page.
        const selection: string[] =
            checkedClusterIds.length === currentClusters.length
                ? []
                : currentClusters.map(({ id }) => id);
        setCheckedClusterIds(selection);
    }

    // After there is a response, if there are clusters or search filter.
    // Conditionally render a subsequent error in addition to most recent successful respnse.
    return (
        <>
            <PageSection>
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                >
                    <Title headingLevel="h1">Clusters</Title>
                    <Flex
                        direction={{ default: 'row' }}
                        alignItems={{ default: 'alignItemsCenter' }}
                    >
                        {hasReadAccessForAdministration && (
                            <FlexItem>
                                <Link to={clustersDelegatedScanningPath}>
                                    Delegated image scanning
                                </Link>
                            </FlexItem>
                        )}
                        {hasReadAccessForAdministration && (
                            <FlexItem>
                                <Link to={clustersDiscoveredClustersPath}>Discovered clusters</Link>
                            </FlexItem>
                        )}
                        {hasAdminRole && (
                            <FlexItem>
                                <Link to={clustersInitBundlesPath}>Init bundles</Link>
                            </FlexItem>
                        )}
                        {hasAdminRole && (
                            <FlexItem>
                                <Link to={clustersClusterRegistrationSecretsPath}>
                                    Cluster registration secrets
                                </Link>
                            </FlexItem>
                        )}
                        {hasWriteAccessForCluster && (
                            <FlexItem>
                                <MenuDropdown
                                    toggleText="Secure a cluster"
                                    onSelect={onSelectInstallMenuItem}
                                    popperProps={{
                                        position: 'end',
                                    }}
                                >
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
                            </FlexItem>
                        )}
                    </Flex>
                </Flex>
                <Content component="p">View the status of secured cluster services</Content>
            </PageSection>
            <PageSection>
                <Toolbar>
                    <ToolbarContent>
                        <CompoundSearchFilter
                            config={filteredSearchFilterConfig}
                            defaultEntity="Cluster"
                            searchFilter={searchFilter}
                            onSearch={(payload) =>
                                setSearchFilter(updateSearchFilter(searchFilter, payload))
                            }
                        />
                        <ToolbarGroup variant="action-group" align={{ default: 'alignEnd' }}>
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
                        <ToolbarGroup className="pf-v6-u-w-100">
                            <CompoundSearchFilterLabels
                                attributesSeparateFromConfig={[]}
                                config={filteredSearchFilterConfig}
                                onFilterChange={setSearchFilter}
                                searchFilter={searchFilter}
                            />
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
                <ClustersTable
                    centralVersion={metadata.version}
                    clusterIdToRetentionInfo={clusterIdToRetentionInfo}
                    tableState={tableState}
                    selectedClusterIds={checkedClusterIds}
                    onClearFilters={() => setSearchFilter({})}
                    onDeleteCluster={onDeleteHandler}
                    toggleAllClusters={toggleAllClusters}
                    toggleCluster={toggleCluster}
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
