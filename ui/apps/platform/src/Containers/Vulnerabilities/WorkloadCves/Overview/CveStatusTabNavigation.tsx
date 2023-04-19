import React from 'react';
import { gql, useQuery } from '@apollo/client';
import {
    PageSection,
    Card,
    CardBody,
    Divider,
    Toolbar,
    ToolbarItem,
    ToolbarContent,
    Pagination,
    TabsComponent,
} from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import ImagesTableContainer from './ImagesTableContainer';
import DeploymentsTableContainer from './DeploymentsTableContainer';
import CVEsTableContainer from './CVEsTableContainer';
import WorkloadTableToolbar from '../components/WorkloadTableToolbar';
import EntityTypeToggleGroup from '../components/EntityTypeToggleGroup';
import { DynamicTableLabel } from '../components/DynamicIcon';
import { DefaultFilters, entityTabValues, EntityTab } from '../types';
import { parseQuerySearchFilter } from '../searchUtils';
import CveStatusTabs, {
    DeferredCvesTab,
    FalsePositiveCvesTab,
    ObservedCvesTab,
} from '../components/CveStatusTabs';

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
    const [activeEntityTabKey] = useURLStringUnion('entityTab', entityTabValues);
    const { page, perPage, setPage, setPerPage } = useURLPagination(25);
    const isFiltered = getHasSearchApplied(querySearchFilter);

    const { data: countsData } = useQuery(entityTypeCountsQuery, {
        variables: {
            query: getRequestQueryStringForSearchFilter({
                ...querySearchFilter,
            }),
        },
    });

    const tableRowCount = getTableRowCount(countsData, activeEntityTabKey);

    return (
        <CveStatusTabs
            component={TabsComponent.nav}
            className="pf-u-pl-lg pf-u-background-color-100"
        >
            <ObservedCvesTab>
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
            </ObservedCvesTab>
            <DeferredCvesTab isDisabled>deferrals tbd</DeferredCvesTab>
            <FalsePositiveCvesTab isDisabled>False-positives tbd</FalsePositiveCvesTab>
        </CveStatusTabs>
    );
}

export default CveStatusTabNavigation;
