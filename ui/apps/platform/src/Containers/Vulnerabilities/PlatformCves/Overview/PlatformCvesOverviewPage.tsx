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

import PageTitle from 'Components/PageTitle';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied } from 'utils/searchUtils';

import TableEntityToolbar from 'Containers/Vulnerabilities/components/TableEntityToolbar';
import useMap from 'hooks/useMap';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';

import SnoozeCveToggleButton from '../../components/SnoozedCveToggleButton';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';
import { platformEntityTabValues } from '../../types';
import useHasLegacySnoozeAbility from '../../hooks/useHasLegacySnoozeAbility';

import ClustersTable from './ClustersTable';
import CVEsTable from './CVEsTable';
import { usePlatformCveEntityCounts } from './usePlatformCveEntityCounts';

function PlatformCvesOverviewPage() {
    const [activeEntityTabKey] = useURLStringUnion('entityTab', platformEntityTabValues);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    // TODO - Need an equivalent function implementation for filter sanitization for Platform CVEs
    const querySearchFilter = searchFilter;
    const isFiltered = getHasSearchApplied(querySearchFilter);

    const isViewingSnoozedCves = querySearchFilter['CVE Snoozed'] === 'true';
    const hasLegacySnoozeAbility = useHasLegacySnoozeAbility();
    const selectedCves = useMap<string, { cve: string }>();

    function onEntityTabChange() {
        pagination.setPage(1);
        // TODO - set default sort here
    }

    const { data } = usePlatformCveEntityCounts(querySearchFilter);

    const entityCounts = {
        CVE: data?.platformCVECount ?? 0,
        Cluster: data?.clusterCount ?? 0,
    };

    const filterToolbar = <></>;

    const entityToggleGroup = (
        <EntityTypeToggleGroup
            entityTabs={['CVE', 'Cluster']}
            entityCounts={entityCounts}
            onChange={onEntityTabChange}
        />
    );

    return (
        <>
            <PageTitle title="Platform CVEs Overview" />
            <Divider component="div" />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-row pf-v5-u-align-items-center"
                variant="light"
            >
                <Flex alignItems={{ default: 'alignItemsCenter' }} className="pf-v5-u-flex-grow-1">
                    <Flex direction={{ default: 'column' }} className="pf-v5-u-flex-grow-1">
                        <Title headingLevel="h1">Platform CVEs</Title>
                        <FlexItem>Prioritize and manage scanned CVEs across clusters</FlexItem>
                    </Flex>
                    <FlexItem>
                        <SnoozeCveToggleButton
                            searchFilter={querySearchFilter}
                            setSearchFilter={setSearchFilter}
                        />
                    </FlexItem>
                </Flex>
            </PageSection>
            <PageSection isCenterAligned isFilled>
                <Card>
                    <CardBody>
                        <TableEntityToolbar
                            filterToolbar={filterToolbar}
                            entityToggleGroup={entityToggleGroup}
                            pagination={pagination}
                            tableRowCount={
                                activeEntityTabKey === 'CVE'
                                    ? entityCounts.CVE
                                    : entityCounts.Cluster
                            }
                            isFiltered={isFiltered}
                        >
                            {hasLegacySnoozeAbility && (
                                <ToolbarItem align={{ default: 'alignRight' }}>
                                    <BulkActionsDropdown isDisabled={selectedCves.size === 0}>
                                        <DropdownItem
                                            key="bulk-snooze-cve"
                                            component="button"
                                            onClick={() => {
                                                // TODO
                                            }}
                                        >
                                            {isViewingSnoozedCves ? 'Unsnooze CVEs' : 'Snooze CVEs'}
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
                                createRowActions={() => []}
                            />
                        )}
                        {activeEntityTabKey === 'Cluster' && (
                            <ClustersTable
                                querySearchFilter={querySearchFilter}
                                isFiltered={isFiltered}
                                pagination={pagination}
                            />
                        )}
                    </CardBody>
                </Card>
            </PageSection>
        </>
    );
}

export default PlatformCvesOverviewPage;
