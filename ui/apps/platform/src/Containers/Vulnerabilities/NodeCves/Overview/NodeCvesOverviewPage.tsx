import React from 'react';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Card,
    CardBody,
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
import { getHasSearchApplied } from 'utils/searchUtils';

import SnoozeCveToggleButton from '../../components/SnoozedCveToggleButton';
import SnoozeCvesModal from '../../components/SnoozeCvesModal/SnoozeCvesModal';
import useSnoozeCveModal from '../../components/SnoozeCvesModal/useSnoozeCveModal';
import useHasLegacySnoozeAbility from '../../hooks/useHasLegacySnoozeAbility';
import TableEntityToolbar from '../../components/TableEntityToolbar';
import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';
import NodeCveFilterToolbar from '../components/NodeCveFilterToolbar';
import { NODE_CVE_SEARCH_OPTION, SNOOZED_NODE_CVE_SEARCH_OPTION } from '../../searchOptions';
import { nodeEntityTabValues } from '../../types';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';

import CVEsTable, {
    sortFields as cveSortFields,
    defaultSortOption as cveDefaultSortOption,
} from './CVEsTable';
import NodesTable, {
    sortFields as nodeSortFields,
    defaultSortOption as nodeDefaultSortOption,
} from './NodesTable';
import { useNodeCveEntityCounts } from './useNodeCveEntityCounts';

const searchOptions = [NODE_CVE_SEARCH_OPTION, SNOOZED_NODE_CVE_SEARCH_OPTION];

function NodeCvesOverviewPage() {
    const apolloClient = useApolloClient();

    const [activeEntityTabKey] = useURLStringUnion('entityTab', nodeEntityTabValues);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { sortOption, getSortParams, setSortOption } = useURLSort({
        sortFields: activeEntityTabKey === 'CVE' ? cveSortFields : nodeSortFields,
        defaultSortOption:
            activeEntityTabKey === 'CVE' ? cveDefaultSortOption : nodeDefaultSortOption,
        onSort: () => pagination.setPage(1, 'replace'),
    });

    // TODO - Need an equivalent function implementation for filter sanitization for Node CVEs
    const querySearchFilter = searchFilter;
    const isFiltered = getHasSearchApplied(querySearchFilter);

    const isViewingSnoozedCves = querySearchFilter['CVE Snoozed']?.[0] === 'true';
    const hasLegacySnoozeAbility = useHasLegacySnoozeAbility();
    const selectedCves = useMap<string, { cve: string }>();
    const { snoozeModalOptions, setSnoozeModalOptions, snoozeActionCreator } = useSnoozeCveModal();

    function onEntityTabChange(entityTab: 'CVE' | 'Node') {
        pagination.setPage(1);
        setSortOption(
            entityTab === 'CVE' ? cveDefaultSortOption : nodeDefaultSortOption,
            'replace'
        );
    }

    const { data } = useNodeCveEntityCounts(querySearchFilter);

    const entityCounts = {
        CVE: data?.nodeCVECount ?? 0,
        Node: data?.nodeCount ?? 0,
    };

    const filterToolbar = (
        <NodeCveFilterToolbar
            searchOptions={searchOptions}
            onFilterChange={() => pagination.setPage(1)}
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
                    onSuccess={() => {
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
                            searchFilter={querySearchFilter}
                            setSearchFilter={setSearchFilter}
                        />
                    </FlexItem>
                </Flex>
            </PageSection>
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
                                />
                            )}
                            {activeEntityTabKey === 'Node' && (
                                <NodesTable
                                    querySearchFilter={querySearchFilter}
                                    isFiltered={isFiltered}
                                    pagination={pagination}
                                    sortOption={sortOption}
                                    getSortParams={getSortParams}
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
