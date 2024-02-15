import React, { useCallback, useEffect, useState } from 'react';
import {
    Bullseye,
    Button,
    ButtonVariant,
    Pagination,
    SearchInput,
    Spinner,
    Text,
    TextVariants,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import omit from 'lodash/omit';

import IconText from 'Components/PatternFly/IconText/IconText';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import {
    ComplianceCheckStatus,
    ComplianceCheckResult,
    ClusterCheckStatus,
    getSingleClusterResultsByScanConfig,
    getSingleClusterResultsByScanConfigCount,
} from 'services/ComplianceEnhancedService';
import { SearchFilter } from 'types/search';
import { SortOption } from 'types/table';
import { searchValueAsArray } from 'utils/searchUtils';

// TODO: move to a shared location
import TableErrorComponent from 'Containers/Vulnerabilities/WorkloadCves/components/TableErrorComponent';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { getClusterResultsStatusObject } from '../compliance.coverage.utils';
import CheckStatusDropdown from '../Components/CheckStatusDropdown';
import CheckStatusModal from '../Components/CheckStatusModal';

type ClusterDetailsTableProps = {
    clusterId: string;
    scanName: string;
};

const sortFields = ['Compliance Check Name'];
const defaultSortOption = {
    field: 'Compliance Check Name',
    direction: 'asc',
} as SortOption;

function ClusterDetailsTable({
    clusterId,
    scanName,
}: ClusterDetailsTableProps): React.ReactElement {
    const [searchCheckValue, setSearchCheckValue] = useState('');
    const [selectedCheckResult, setSelectedCheckResult] = useState<ComplianceCheckResult | null>(
        null
    );
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
    });

    const listQuery = useCallback(
        () =>
            getSingleClusterResultsByScanConfig(
                clusterId,
                scanName,
                searchFilter,
                sortOption,
                page - 1,
                perPage
            ),
        [clusterId, scanName, searchFilter, sortOption, page, perPage]
    );
    const { data: scanResults, loading: isLoading, error } = useRestQuery(listQuery);

    const countQuery = useCallback(
        () => getSingleClusterResultsByScanConfigCount(clusterId, scanName, searchFilter),
        [clusterId, scanName, searchFilter]
    );
    const { data: scanResultsCount } = useRestQuery(countQuery);

    useEffect(() => {
        const checkNameFilter = searchFilter['Compliance Check Name'];

        if (typeof checkNameFilter === 'string') {
            setSearchCheckValue(checkNameFilter);
        } else {
            setSearchCheckValue('');
        }
    }, [searchFilter]);

    function getMatchingCluster(clusters: ClusterCheckStatus[]): ClusterCheckStatus | null {
        return (
            clusters.find(
                (clusterCheckStatus) => clusterCheckStatus.cluster.clusterId === clusterId
            ) ?? null
        );
    }

    function getStatusByClusterId(clusters: ClusterCheckStatus[]): ComplianceCheckStatus | null {
        const matchingCluster = getMatchingCluster(clusters);
        return matchingCluster ? matchingCluster.status : null;
    }

    function onChangeSearchFilter(newFilter: SearchFilter) {
        setSearchFilter(newFilter);
    }

    function onSearchInputChange(_event, value) {
        setSearchCheckValue(value);
    }

    const handleCheckInputSearch = () => {
        const newFilter = {
            ...searchFilter,
            'Compliance Check Name': searchCheckValue,
        };
        setSearchFilter(newFilter);
    };

    const handleCheckInputClear = () => {
        const newFilters = omit(searchFilter, 'Compliance Check Name');
        setSearchCheckValue('');
        setSearchFilter(newFilters);
    };

    function onSelect(type: 'Compliance Check Status', checked: boolean, selection: string) {
        const selectedSearchFilter = searchValueAsArray(searchFilter[type]);
        onChangeSearchFilter({
            ...searchFilter,
            [type]: checked
                ? [...selectedSearchFilter, selection]
                : selectedSearchFilter.filter((value) => value !== selection),
        });
    }

    const renderTableContent = () => {
        return scanResults?.checkResults.map((checkResult) => {
            const { checkName, rationale, clusters } = checkResult;
            const status = getStatusByClusterId(clusters);
            const statusObj = status ? getClusterResultsStatusObject(status) : null;

            return (
                <Tr key={checkName}>
                    <Td modifier="truncate">
                        <Text>{checkName}</Text>
                        <Text component={TextVariants.small} className="pf-u-color-200">
                            {rationale}
                        </Text>
                    </Td>
                    <Td>
                        {statusObj && (
                            <Button
                                isInline
                                variant={ButtonVariant.link}
                                onClick={() => setSelectedCheckResult(checkResult)}
                            >
                                <IconText icon={statusObj.icon} text={statusObj.statusText} />
                            </Button>
                        )}
                    </Td>
                </Tr>
            );
        });
    };

    const renderLoadingContent = () => (
        <Tr>
            <Td colSpan={2}>
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            </Td>
        </Tr>
    );

    const renderErrorContent = () => {
        if (error) {
            return (
                <Tr>
                    <Td colSpan={2}>
                        <TableErrorComponent
                            error={error}
                            message="An error occurred. Try refreshing again"
                        />
                    </Td>
                </Tr>
            );
        }
        return <></>;
    };

    const renderEmptyContent = () => (
        <Tr>
            <Td colSpan={2}>
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
        if (scanResults && scanResults.checkResults.length > 0) {
            return renderTableContent();
        }
        if (error) {
            return renderErrorContent();
        }
        return renderEmptyContent();
    };

    return (
        <>
            <Title headingLevel="h2" className="pf-u-px-md">
                {scanName}
            </Title>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem variant="search-filter" className="pf-u-flex-grow-1">
                        <SearchInput
                            aria-label="Filter results by check"
                            placeholder="Filter results by check"
                            value={searchCheckValue}
                            onChange={onSearchInputChange}
                            onSearch={handleCheckInputSearch}
                            onClear={handleCheckInputClear}
                        />
                    </ToolbarItem>
                    <ToolbarItem>
                        <CheckStatusDropdown searchFilter={searchFilter} onSelect={onSelect} />
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                        <Pagination
                            itemCount={scanResultsCount ?? 0}
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
                        <Th width={90} sort={getSortParams('Compliance Check Name')}>
                            Compliance check
                        </Th>
                        <Th>Status</Th>
                    </Tr>
                </Thead>
                <Tbody>{renderTableBodyContent()}</Tbody>
            </TableComposable>

            {selectedCheckResult && (
                <CheckStatusModal
                    checkResult={selectedCheckResult}
                    status={getStatusByClusterId(selectedCheckResult.clusters)}
                    isOpen
                    handleClose={() => setSelectedCheckResult(null)}
                />
            )}
        </>
    );
}

export default ClusterDetailsTable;
