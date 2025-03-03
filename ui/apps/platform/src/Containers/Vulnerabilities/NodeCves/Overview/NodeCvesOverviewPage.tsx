import React, { useEffect } from 'react';
import {
    Alert,
    Card,
    CardBody,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Title,
    ToolbarItem,
} from '@patternfly/react-core';
import { DropdownItem } from '@patternfly/react-core/deprecated';
import { useApolloClient } from '@apollo/client';

import PageTitle from 'Components/PageTitle';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import useMap from 'hooks/useMap';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import useAnalytics, {
    GLOBAL_SNOOZE_CVE,
    NODE_CVE_ENTITY_CONTEXT_VIEWED,
    NODE_CVE_FILTER_APPLIED,
} from 'hooks/useAnalytics';
import { getHasSearchApplied } from 'utils/searchUtils';

import { parseQuerySearchFilter } from 'Containers/Vulnerabilities/utils/searchUtils';
import useSnoozedCveCount from 'Containers/Vulnerabilities/hooks/useSnoozedCveCount';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import {
    clusterSearchFilterConfig,
    nodeComponentSearchFilterConfig,
    nodeCVESearchFilterConfig,
    nodeSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import SnoozeCveToggleButton from '../../components/SnoozedCveToggleButton';
import SnoozeCvesModal from '../../components/SnoozeCvesModal/SnoozeCvesModal';
import useSnoozeCveModal from '../../components/SnoozeCvesModal/useSnoozeCveModal';
import useHasLegacySnoozeAbility from '../../hooks/useHasLegacySnoozeAbility';
import TableEntityToolbar from '../../components/TableEntityToolbar';
import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';
import { nodeEntityTabValues } from '../../types';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';

import CVEsTable, {
    defaultSortOption as cveDefaultSortOption,
    sortFields as cveSortFields,
} from './CVEsTable';
import NodesTable, {
    defaultSortOption as nodeDefaultSortOption,
    sortFields as nodeSortFields,
} from './NodesTable';
import { useNodeCveEntityCounts } from './useNodeCveEntityCounts';
import ExternalLink from '../../../../Components/PatternFly/IconText/ExternalLink';
import { getVersionedDocs } from '../../../../utils/versioning';
import useMetadata from '../../../../hooks/useMetadata';
import useFeatureFlags from '../../../../hooks/useFeatureFlags';

const searchFilterConfig = [
    nodeSearchFilterConfig,
    nodeCVESearchFilterConfig,
    nodeComponentSearchFilterConfig,
    clusterSearchFilterConfig,
];

function NodeCvesOverviewPage() {
    const apolloClient = useApolloClient();
    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const scannerV4NodeScanResultsPossible =
        isFeatureFlagEnabled('ROX_SCANNER_V4') && isFeatureFlagEnabled('ROX_NODE_INDEX_ENABLED');

    const [activeEntityTabKey] = useURLStringUnion('entityTab', nodeEntityTabValues);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { sortOption, getSortParams, setSortOption } = useURLSort({
        sortFields: activeEntityTabKey === 'CVE' ? cveSortFields : nodeSortFields,
        defaultSortOption:
            activeEntityTabKey === 'CVE' ? cveDefaultSortOption : nodeDefaultSortOption,
        onSort: () => pagination.setPage(1),
    });

    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);

    const isViewingSnoozedCves = querySearchFilter['CVE Snoozed']?.[0] === 'true';
    const hasLegacySnoozeAbility = useHasLegacySnoozeAbility();
    const selectedCves = useMap<string, { cve: string }>();
    const { snoozeModalOptions, setSnoozeModalOptions, snoozeActionCreator } = useSnoozeCveModal();
    const snoozedCveCount = useSnoozedCveCount('Node');
    const { version } = useMetadata();

    function onEntityTabChange(entityTab: 'CVE' | 'Node') {
        pagination.setPage(1);
        setSortOption(entityTab === 'CVE' ? cveDefaultSortOption : nodeDefaultSortOption);

        analyticsTrack({
            event: NODE_CVE_ENTITY_CONTEXT_VIEWED,
            properties: {
                type: entityTab,
                page: 'Overview',
            },
        });
    }

    // Track the current entity tab when the page is initially visited.
    useEffect(() => {
        onEntityTabChange(activeEntityTabKey);
    }, []);

    function onClearFilters() {
        setSearchFilter({});
        pagination.setPage(1);
    }

    const { data } = useNodeCveEntityCounts(querySearchFilter);

    const entityCounts = {
        CVE: data?.nodeCVECount ?? 0,
        Node: data?.nodeCount ?? 0,
    };

    const filterToolbar = (
        <AdvancedFiltersToolbar
            searchFilter={searchFilter}
            searchFilterConfig={searchFilterConfig}
            onFilterChange={(newFilter, searchPayload) => {
                setSearchFilter(newFilter);
                pagination.setPage(1);
                trackAppliedFilter(NODE_CVE_FILTER_APPLIED, searchPayload);
            }}
        />
    );

    const entityToggleGroup = (
        <EntityTypeToggleGroup
            entityTabs={['CVE', 'Node']}
            entityCounts={entityCounts}
            onChange={onEntityTabChange}
        />
    );

    return (
        <>
            {snoozeModalOptions && (
                <SnoozeCvesModal
                    {...snoozeModalOptions}
                    onSuccess={(action, duration) => {
                        if (action === 'SNOOZE') {
                            analyticsTrack({
                                event: GLOBAL_SNOOZE_CVE,
                                properties: { type: 'NODE', duration },
                            });
                        }
                        // Refresh the data after snoozing/unsnoozing CVEs
                        apolloClient.cache.evict({ fieldName: 'nodeCVEs' });
                        apolloClient.cache.evict({ fieldName: 'nodeCVECount' });
                        apolloClient.cache.gc();
                        selectedCves.clear();
                    }}
                    onClose={() => setSnoozeModalOptions(null)}
                />
            )}
            <PageTitle title="Node CVEs Overview" />
            <Divider component="div" />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-row pf-v5-u-align-items-center"
                variant="light"
            >
                <Flex alignItems={{ default: 'alignItemsCenter' }} className="pf-v5-u-flex-grow-1">
                    <Flex direction={{ default: 'column' }} className="pf-v5-u-flex-grow-1">
                        <Title headingLevel="h1">Node CVEs</Title>
                        <FlexItem>Prioritize and manage scanned CVEs across nodes</FlexItem>
                    </Flex>
                    <FlexItem>
                        <SnoozeCveToggleButton
                            searchFilter={searchFilter}
                            setSearchFilter={setSearchFilter}
                            snoozedCveCount={snoozedCveCount}
                        />
                    </FlexItem>
                </Flex>
            </PageSection>
            {scannerV4NodeScanResultsPossible && (
                <PageSection variant="light" className="pf-v5-u-pt-0">
                    <Alert
                        isInline
                        variant="info"
                        title="Results may include Node CVEs obtained from Scanner V4"
                        component="p"
                    >
                        <ExternalLink>
                            <a
                                href={getVersionedDocs(
                                    version,
                                    'operating/managing-vulnerabilities#understanding-node-cves-scanner-v4_scan-rhcos-node-host'
                                )}
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                Read more about the differences between the node scanning results
                                obtained with the StackRox Scanner and Scanner V4.
                            </a>
                        </ExternalLink>
                    </Alert>
                </PageSection>
            )}
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            <TableEntityToolbar
                                filterToolbar={filterToolbar}
                                entityToggleGroup={entityToggleGroup}
                                pagination={pagination}
                                tableRowCount={
                                    activeEntityTabKey === 'CVE'
                                        ? entityCounts.CVE
                                        : entityCounts.Node
                                }
                                isFiltered={isFiltered}
                            >
                                {hasLegacySnoozeAbility && (
                                    <ToolbarItem align={{ default: 'alignRight' }}>
                                        <BulkActionsDropdown isDisabled={selectedCves.size === 0}>
                                            <DropdownItem
                                                key="bulk-snooze-cve"
                                                component="button"
                                                onClick={() =>
                                                    setSnoozeModalOptions({
                                                        action: isViewingSnoozedCves
                                                            ? 'UNSNOOZE'
                                                            : 'SNOOZE',
                                                        cveType: 'NODE_CVE',
                                                        cves: Array.from(selectedCves.values()),
                                                    })
                                                }
                                            >
                                                {isViewingSnoozedCves
                                                    ? 'Unsnooze CVEs'
                                                    : 'Snooze CVEs'}
                                            </DropdownItem>
                                        </BulkActionsDropdown>
                                    </ToolbarItem>
                                )}
                            </TableEntityToolbar>
                            <Divider component="div" />
                            {activeEntityTabKey === 'CVE' && (
                                <CVEsTable
                                    querySearchFilter={querySearchFilter}
                                    isFiltered={isFiltered}
                                    pagination={pagination}
                                    selectedCves={selectedCves}
                                    canSelectRows={hasLegacySnoozeAbility}
                                    createRowActions={snoozeActionCreator(
                                        'NODE_CVE',
                                        isViewingSnoozedCves ? 'UNSNOOZE' : 'SNOOZE'
                                    )}
                                    sortOption={sortOption}
                                    getSortParams={getSortParams}
                                    onClearFilters={onClearFilters}
                                />
                            )}
                            {activeEntityTabKey === 'Node' && (
                                <NodesTable
                                    querySearchFilter={querySearchFilter}
                                    isFiltered={isFiltered}
                                    pagination={pagination}
                                    sortOption={sortOption}
                                    getSortParams={getSortParams}
                                    onClearFilters={onClearFilters}
                                />
                            )}
                        </CardBody>
                    </Card>
                </PageSection>
            </PageSection>
        </>
    );
}

export default NodeCvesOverviewPage;
