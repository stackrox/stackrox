import React, { useCallback } from 'react';
import { generatePath, Link } from 'react-router-dom';
import {
    Alert,
    Bullseye,
    Flex,
    FlexItem,
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
import { CubesIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import {
    complianceEnhancedCoverageClustersPath,
    complianceEnhancedScanConfigsPath,
} from 'routePaths';
import { getAllClustersCombinedStats } from 'services/ComplianceEnhancedService';

import ComplianceClusterStatus from '../ScanConfigs/components/ComplianceClusterStatus';
import CoverageTableViewToggleGroup from './Components/CoverageTableViewToggleGroup';

import {
    calculateCompliancePercentage,
    getCompliancePfClassName,
    getStatusCounts,
} from './compliance.coverage.utils';

function ClustersCoverageTable() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);

    const listQuery = useCallback(
        () => getAllClustersCombinedStats(page - 1, perPage),
        [page, perPage]
    );
    const { data: clusterScanStats, loading: isLoading, error } = useRestQuery(listQuery);

    const renderTableContent = () => {
        return clusterScanStats?.map(({ cluster, checkStats, clusterErrors }, index) => {
            const { passCount, totalCount } = getStatusCounts(checkStats);
            const passPercentage = calculateCompliancePercentage(passCount, totalCount);

            return (
                <Tr key={cluster.clusterId}>
                    <Td dataLabel="Cluster">
                        <Link
                            to={generatePath(complianceEnhancedCoverageClustersPath, {
                                clusterId: cluster.clusterId,
                            })}
                        >
                            {cluster.clusterName}
                        </Link>
                    </Td>
                    <Td dataLabel="Operator status">
                        <ComplianceClusterStatus errors={clusterErrors} />
                    </Td>
                    <Td dataLabel="Compliance">
                        <Progress
                            id={`progress-bar-${index}`}
                            value={passPercentage}
                            measureLocation={ProgressMeasureLocation.outside}
                            className={getCompliancePfClassName(passPercentage)}
                            aria-label={`${cluster.clusterName} compliance percentage`}
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
            <Td colSpan={3}>
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            </Td>
        </Tr>
    );

    const renderEmptyContent = () => (
        <Tr>
            <Td colSpan={3}>
                <Bullseye>
                    <EmptyStateTemplate
                        title="No scan data available"
                        headingLevel="h2"
                        icon={CubesIcon}
                    >
                        <Flex direction={{ default: 'column' }}>
                            <FlexItem>
                                <Text>
                                    Schedule a scan to view results. If you have already configured
                                    a scan to run, then please check back later for page results.
                                </Text>
                            </FlexItem>
                            <FlexItem>
                                <Link to={complianceEnhancedScanConfigsPath}>
                                    Go to scan schedules
                                </Link>
                            </FlexItem>
                        </Flex>
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
                            <ToolbarItem>
                                <CoverageTableViewToggleGroup />
                            </ToolbarItem>
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
