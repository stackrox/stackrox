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
import NodeCveFilterToolbar from '../components/NodeCveFilterToolbar';
import { NODE_CVE_SEARCH_OPTION } from '../../searchOptions';
import { parseWorkloadQuerySearchFilter } from '../../utils/searchUtils';
import { nodeEntityTabValues } from '../../types';

import CVEsTableContainer from './CVEsTableContainer';
import NodesTableContainer from './NodesTableContainer';

const searchOptions = [NODE_CVE_SEARCH_OPTION];

function NodeCvesOverviewPage() {
    const [activeEntityTabKey] = useURLStringUnion('entityTab', nodeEntityTabValues);
    const { searchFilter } = useURLSearch();
    const pagination = useURLPagination(20);

    // TODO - Need an equivalent function implementation for filter sanitization for Node CVEs
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);

    function onEntityTabChange() {
        pagination.setPage(1);
        // TODO - set default sort here
    }

    // TODO - needs to be connected to a query
    const entityCounts = {
        CVE: 0,
        Node: 0,
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
            <PageTitle title="Node CVEs Overview" />
            <Divider component="div" />
            <PageSection
                className="pf-u-display-flex pf-u-flex-direction-row pf-u-align-items-center"
                variant="light"
            >
                <Flex direction={{ default: 'column' }} className="pf-u-flex-grow-1">
                    <Title headingLevel="h1">Node CVEs</Title>
                    <FlexItem>Prioritize and manage scanned CVEs across nodes</FlexItem>
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
                            {activeEntityTabKey === 'Node' && (
                                <NodesTableContainer
                                    filterToolbar={filterToolbar}
                                    entityToggleGroup={entityToggleGroup}
                                    querySearchFilter={querySearchFilter}
                                    isFiltered={isFiltered}
                                    rowCount={entityCounts.Node}
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

export default NodeCvesOverviewPage;
