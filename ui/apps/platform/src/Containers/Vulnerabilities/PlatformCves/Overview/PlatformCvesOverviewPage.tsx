import React from 'react';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Card,
    CardBody,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied } from 'utils/searchUtils';

import TableEntityToolbar from 'Containers/Vulnerabilities/components/TableEntityToolbar';
import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';
import { parseWorkloadQuerySearchFilter } from '../../utils/searchUtils';
import { platformEntityTabValues } from '../../types';

import ClustersTable from './ClustersTable';
import CVEsTable from './CVEsTable';

function PlatformCvesOverviewPage() {
    const [activeEntityTabKey] = useURLStringUnion('entityTab', platformEntityTabValues);
    const { searchFilter } = useURLSearch();
    const pagination = useURLPagination(20);

    // TODO - Need an equivalent function implementation for filter sanitization for Platform CVEs
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);

    function onEntityTabChange() {
        pagination.setPage(1);
        // TODO - set default sort here
    }

    // TODO - needs to be connected to a query
    const entityCounts = {
        CVE: 0,
        Cluster: 0,
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
                <Flex direction={{ default: 'column' }} className="pf-v5-u-flex-grow-1">
                    <Title headingLevel="h1">Platform CVEs</Title>
                    <FlexItem>Prioritize and manage scanned CVEs across clusters</FlexItem>
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
                        />
                        <Divider component="div" />
                        {activeEntityTabKey === 'CVE' && (
                            <CVEsTable
                                querySearchFilter={querySearchFilter}
                                isFiltered={isFiltered}
                                pagination={pagination}
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
