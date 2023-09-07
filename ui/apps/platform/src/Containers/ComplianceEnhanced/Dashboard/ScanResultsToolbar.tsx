import React, { useState } from 'react';
import {
    Pagination,
    SearchInput,
    SelectOption,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { SearchFilter } from 'types/search';

type ScanResultsToolbarProps = {
    numberOfScanResults: number | null;
    searchFilter: SearchFilter;
    setSearchFilter: (searchFilter: SearchFilter) => void;
} & UseURLPaginationResult;

function ScanResultsToolbar({
    numberOfScanResults,
    searchFilter,
    setSearchFilter,
    page,
    perPage,
    setPage,
    setPerPage,
}: ScanResultsToolbarProps) {
    const [scanFilter, setScanFilter] = useState<string>('');
    const [profileFilter, setProfileFilter] = useState<string>('');
    const [clusterFilter, setClusterFilter] = useState<string>('');
    const [searchType, setSearchType] = useState<string>('Profile');

    const resetSearchFilterKey = (key: keyof SearchFilter) => {
        const currentFilter = searchFilter;
        delete currentFilter[key];
        setSearchFilter({
            ...currentFilter,
        });
    };

    function onSearchTypeSelect(_e, selection) {
        switch (selection) {
            case 'Profile':
                resetSearchFilterKey('Cluster Name');
                setClusterFilter('');
                break;
            case 'Cluster':
                resetSearchFilterKey('Profile Name');
                setProfileFilter('');
                break;
            default:
                break;
        }
        setSearchType(selection);
    }

    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarItem variant="search-filter">
                    <SearchInput
                        aria-label="Filter by scan name"
                        placeholder="Filter by scan name"
                        value={scanFilter}
                        onChange={(_e, selection: string) => setScanFilter(selection)}
                        onSearch={() =>
                            setSearchFilter({ ...searchFilter, 'Scan Name': scanFilter })
                        }
                        onClear={() => {
                            setScanFilter('');
                            resetSearchFilterKey('Scan Name');
                        }}
                    />
                </ToolbarItem>
                <ToolbarItem variant="separator" />
                {searchType === 'Profile' ? (
                    <ToolbarItem variant="search-filter">
                        <SearchInput
                            aria-label="Filter by profile name"
                            placeholder="Filter by profile name"
                            value={profileFilter}
                            onChange={(_e, selection: string) => setProfileFilter(selection)}
                            onSearch={() =>
                                setSearchFilter({ ...searchFilter, 'Profile Name': profileFilter })
                            }
                            onClear={() => {
                                setProfileFilter('');
                                resetSearchFilterKey('Profile Name');
                            }}
                        />
                    </ToolbarItem>
                ) : (
                    <ToolbarItem variant="search-filter">
                        <SearchInput
                            aria-label="Filter by cluster name"
                            placeholder="Filter by cluster name"
                            value={clusterFilter}
                            onChange={(_e, selection: string) => setClusterFilter(selection)}
                            onSearch={() =>
                                setSearchFilter({ ...searchFilter, 'Cluster Name': clusterFilter })
                            }
                            onClear={() => {
                                setClusterFilter('');
                                resetSearchFilterKey('Cluster Name');
                            }}
                        />
                    </ToolbarItem>
                )}
                <ToolbarItem variant="search-filter">
                    <SelectSingle
                        id="scan-results-filter-type"
                        value={searchType}
                        handleSelect={onSearchTypeSelect}
                    >
                        <SelectOption value="Profile" />
                        <SelectOption value="Cluster" />
                    </SelectSingle>
                </ToolbarItem>
                {numberOfScanResults && (
                    <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                        <Pagination
                            isCompact
                            itemCount={numberOfScanResults}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                        />
                    </ToolbarItem>
                )}
            </ToolbarContent>
        </Toolbar>
    );
}

export default ScanResultsToolbar;
