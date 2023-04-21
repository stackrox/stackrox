import React from 'react';
import { gql, useQuery } from '@apollo/client';
import {
    Tabs,
    Tab,
    TabTitleText,
    TabsComponent,
    PageSection,
    Card,
    CardBody,
    Divider,
    Toolbar,
    ToolbarItem,
    ToolbarContent,
    Pagination,
} from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import ImagesTableContainer from './ImagesTableContainer';
import DeploymentsTableContainer from './DeploymentsTableContainer';
import CVEsTableContainer from './CVEsTableContainer';
import WorkloadTableToolbar from '../WorkloadTableToolbar';
import EntityTypeToggleGroup from '../components/EntityTypeToggleGroup';
import { DynamicTableLabel } from '../components/DynamicIcon';
import { DefaultFilters, cveStatusTabValues, entityTabValues, EntityTab } from '../types';
import { parseQuerySearchFilter } from '../searchUtils';

type CveStatusTabNavigationProps = {
    defaultFilters: DefaultFilters;
};

type EntityCounts = {
    imageCount: number;
    deploymentCount: number;
    imageCVECount: number;
};

const entityTypeCountsQuery = gql`
    query getEntityTypeCounts($query: String) {
        imageCount(query: $query)
        deploymentCount(query: $query)
        imageCVECount(query: $query)
    }
`;

function getTableRowCount(countsData: EntityCounts, entityType: EntityTab): number {
    switch (entityType) {
        case 'Image':
            return countsData?.imageCount;
        case 'Deployment':
            return countsData?.deploymentCount;
        case 'CVE':
            return countsData?.imageCVECount;
        default:
            return 0;
    }
}

function CveStatusTabNavigation({ defaultFilters }: CveStatusTabNavigationProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const [activeCVEStatusKey, setActiveCVEStatusKey] = useURLStringUnion(
        'cveStatus',
        cveStatusTabValues
    );
    const [activeEntityTabKey] = useURLStringUnion('entityTab', entityTabValues);
    const { page, perPage, setPage, setPerPage } = useURLPagination(25);
    const isFiltered = getHasSearchApplied(querySearchFilter);

    function handleTabClick(e, tab) {
        setActiveCVEStatusKey(tab);
    }

    const { data: countsData } = useQuery(entityTypeCountsQuery, {
        variables: {
            query: getRequestQueryStringForSearchFilter({
                ...querySearchFilter,
            }),
        },
    });

    const tableRowCount = getTableRowCount(countsData, activeEntityTabKey);

    return (
        <Tabs
            activeKey={activeCVEStatusKey}
            onSelect={handleTabClick}
            component={TabsComponent.nav}
            className="pf-u-pl-lg pf-u-background-color-100"
            mountOnEnter
            unmountOnExit
        >
            <Tab eventKey="Observed" title={<TabTitleText>Observed CVEs</TabTitleText>}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            <WorkloadTableToolbar defaultFilters={defaultFilters} />
                            <Divider component="div" />
                            <Toolbar>
                                <ToolbarContent>
                                    <ToolbarItem>
                                        <EntityTypeToggleGroup
                                            imageCount={countsData?.imageCount}
                                            cveCount={countsData?.cveCount}
                                            deploymentCount={countsData?.deploymentCount}
                                        />
                                    </ToolbarItem>
                                    {isFiltered && (
                                        <ToolbarItem>
                                            <DynamicTableLabel />
                                        </ToolbarItem>
                                    )}
                                    <ToolbarItem
                                        alignment={{ default: 'alignRight' }}
                                        variant="pagination"
                                    >
                                        <Pagination
                                            isCompact
                                            itemCount={tableRowCount}
                                            page={page}
                                            perPage={perPage}
                                            onSetPage={(_, newPage) => setPage(newPage)}
                                            onPerPageSelect={(_, newPerPage) => {
                                                if (tableRowCount < (page - 1) * newPerPage) {
                                                    setPage(1);
                                                }
                                                setPerPage(newPerPage);
                                            }}
                                        />
                                    </ToolbarItem>
                                </ToolbarContent>
                            </Toolbar>
                            <Divider component="div" />
                            {activeEntityTabKey === 'CVE' && <CVEsTableContainer />}
                            {activeEntityTabKey === 'Image' && <ImagesTableContainer />}
                            {activeEntityTabKey === 'Deployment' && <DeploymentsTableContainer />}
                        </CardBody>
                    </Card>
                </PageSection>
            </Tab>
            <Tab eventKey="Deferred" title={<TabTitleText>Deferrals</TabTitleText>} isDisabled>
                deferrals tbd
            </Tab>
            <Tab
                eventKey="False Positive"
                title={<TabTitleText>False Positives</TabTitleText>}
                isDisabled
            >
                False-positives tbd
            </Tab>
        </Tabs>
    );
}

export default CveStatusTabNavigation;
