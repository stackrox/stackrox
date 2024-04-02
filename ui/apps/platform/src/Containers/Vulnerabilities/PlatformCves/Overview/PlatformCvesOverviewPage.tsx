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

import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';
import { parseWorkloadQuerySearchFilter } from '../../utils/searchUtils';
import { platformEntityTabValues } from '../../types';

import CVEsTableContainer from './CVEsTableContainer';
import ClustersTableContainer from './ClustersTableContainer';

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
                className="pf-u-display-flex pf-u-flex-direction-row pf-u-align-items-center"
                variant="light"
            >
                <Flex direction={{ default: 'column' }} className="pf-u-flex-grow-1">
                    <Title headingLevel="h1">Platform CVEs</Title>
                    <FlexItem>Prioritize and manage scanned CVEs across clusters</FlexItem>
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            {activeEntityTabKey === 'CVE' && (
                                <CVEsTableContainer
                                    filterToolbar={filterToolbar}
                                    entityToggleGroup={entityToggleGroup}
                                    querySearchFilter={querySearchFilter}
                                    isFiltered={isFiltered}
                                    rowCount={entityCounts.CVE}
                                    pagination={pagination}
                                />
                            )}
                            {activeEntityTabKey === 'Cluster' && (
                                <ClustersTableContainer
                                    filterToolbar={filterToolbar}
                                    entityToggleGroup={entityToggleGroup}
                                    querySearchFilter={querySearchFilter}
                                    isFiltered={isFiltered}
                                    rowCount={entityCounts.Cluster}
                                    pagination={pagination}
                                />
                            )}
                        </CardBody>
                    </Card>
                </PageSection>
            </PageSection>
        </>
    );
}

export default PlatformCvesOverviewPage;
