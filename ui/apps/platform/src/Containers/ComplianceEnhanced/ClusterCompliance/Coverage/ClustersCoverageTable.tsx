import React, { useCallback } from 'react';
import { Link } from 'react-router-dom';
import {
    Alert,
    Bullseye,
    Button,
    PageSection,
    Pagination,
    Progress,
    ProgressMeasureLocation,
    Spinner,
    Text,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { TableComposable, Thead, Tr, Th, Td, Tbody } from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';

import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getComplianceClusterScanStats } from 'services/ComplianceEnhancedService';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';

import {
    calculateCompliancePercentage,
    getPassAndTotalCount,
    getProgressBarVariant,
} from './compliance.coverage.utils';

function ClustersCoverageTable() {
    const { setSearchFilter } = useURLSearch();
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);

    const listQuery = useCallback(
        () => getComplianceClusterScanStats(page - 1, perPage),
        [page, perPage]
    );
    const { data: clusterScanStats, loading: isLoading, error } = useRestQuery(listQuery);

    const renderTableContent = () => {
        return clusterScanStats?.map(({ cluster, checkStats }, index) => {
            const { passCount, totalCount } = getPassAndTotalCount(checkStats);
            const passPercentage = calculateCompliancePercentage(passCount, totalCount);

            return (
                <Tr key={cluster.clusterId}>
                    <Td>
                        {/* eslint-disable-next-line jsx-a11y/anchor-is-valid */}
                        <Link to="#">{cluster.clusterName}</Link>
                    </Td>
                    <Td>WIP</Td>
                    <Td>WIP</Td>
                    <Td>
                        <Progress
                            id={`progress-bar-${index}`}
                            value={passPercentage}
                            measureLocation={ProgressMeasureLocation.outside}
                            variant={getProgressBarVariant(passPercentage)}
                        />
                        <Tooltip
                            content={
                                <div>
                                    {`${passCount} / ${totalCount} checks are passing for this cluster`}
                                </div>
                            }
                            reference={() =>
                                document.getElementById(
                                    `progress-bar-${index}`
                                ) as HTMLButtonElement
                            }
                        />
                    </Td>
                </Tr>
            );
        });
    };

    const renderLoadingContent = () => (
        <Tr>
            <Td colSpan={4}>
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            </Td>
        </Tr>
    );

    const renderEmptyContent = () => (
        <Tr>
            <Td colSpan={4}>
                <Bullseye>
                    <EmptyStateTemplate
                        title="No results found"
                        headingLevel="h2"
                        icon={SearchIcon}
                    >
                        <Text>Clear all filters and try again.</Text>
                        <Button variant="link" onClick={() => setSearchFilter({})}>
                            Clear filters
                        </Button>
                    </EmptyStateTemplate>
                </Bullseye>
            </Td>
        </Tr>
    );

    const renderTableBodyContent = () => {
        if (isLoading) {
            return renderLoadingContent();
        }
        if (clusterScanStats && clusterScanStats.length > 0) {
            return renderTableContent();
        }
        return renderEmptyContent();
    };

    return (
        <>
            {error ? (
                <PageSection variant="light" isFilled>
                    <Bullseye>
                        <Alert variant="danger" title={error} />
                    </Bullseye>
                </PageSection>
            ) : (
                <>
                    <Toolbar>
                        <ToolbarContent>
                            <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                                <Pagination
                                    isCompact
                                    itemCount={clusterScanStats ? clusterScanStats.length : 0}
                                    page={page}
                                    perPage={perPage}
                                    onSetPage={(_, newPage) => setPage(newPage)}
                                    onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>

                    <TableComposable>
                        <Thead noWrap>
                            <Tr>
                                <Th>Cluster</Th>
                                <Th>Operator status</Th>
                                <Th>Build date</Th>
                                <Th
                                    info={{
                                        tooltip:
                                            'Percentage of passing checks across scanned profiles',
                                    }}
                                >
                                    Compliance
                                </Th>
                            </Tr>
                        </Thead>
                        <Tbody>{renderTableBodyContent()}</Tbody>
                    </TableComposable>
                </>
            )}
        </>
    );
}

export default ClustersCoverageTable;
