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
import { UsePaginationResult } from 'hooks/patternfly/usePagination';

export type SearchFilter = {
    scanName: string;
    clusterName: string;
    profileName: string;
};

type FilterInputProps = {
    setSearchFilter: React.Dispatch<React.SetStateAction<SearchFilter>>;
};

type ProfileFilterInputProps = FilterInputProps & {
    profileFilter: string;
    setProfileFilter: React.Dispatch<React.SetStateAction<string>>;
};

type ClusterFilterInputProps = FilterInputProps & {
    clusterFilter: string;
    setClusterFilter: React.Dispatch<React.SetStateAction<string>>;
};

function ProfileFilterInput({
    profileFilter,
    setSearchFilter,
    setProfileFilter,
}: ProfileFilterInputProps) {
    return (
        <ToolbarItem variant="search-filter">
            <SearchInput
                aria-label="Filter by profile name"
                placeholder="Filter by profile name"
                value={profileFilter}
                onChange={(_e, selection: string) => setProfileFilter(selection)}
                onSearch={() =>
                    setSearchFilter((prev) => ({ ...prev, profileName: profileFilter }))
                }
                onClear={() => {
                    setProfileFilter('');
                    setSearchFilter((prev) => ({ ...prev, profileName: '' }));
                }}
            />
        </ToolbarItem>
    );
}

function ClusterFilterInput({
    clusterFilter,
    setSearchFilter,
    setClusterFilter,
}: ClusterFilterInputProps) {
    return (
        <ToolbarItem variant="search-filter">
            <SearchInput
                aria-label="Filter by cluster name"
                placeholder="Filter by cluster name"
                value={clusterFilter}
                onChange={(_e, selection: string) => setClusterFilter(selection)}
                onSearch={() =>
                    setSearchFilter((prev) => ({ ...prev, clusterName: clusterFilter }))
                }
                onClear={() => {
                    setClusterFilter('');
                    setSearchFilter((prev) => ({ ...prev, clusterName: '' }));
                }}
            />
        </ToolbarItem>
    );
}

type ScanResultsToolbarProps = {
    numberOfScanResults: number;
    searchFilter: SearchFilter;
    setSearchFilter: React.Dispatch<React.SetStateAction<SearchFilter>>;
} & UsePaginationResult;

function ScanResultsToolbar({
    numberOfScanResults,
    searchFilter,
    setSearchFilter,
    page,
    perPage,
    onSetPage,
    onPerPageSelect,
}: ScanResultsToolbarProps) {
    const [scanFilter, setScanFilter] = useState<string>('');
    const [profileFilter, setProfileFilter] = useState<string>('');
    const [clusterFilter, setClusterFilter] = useState<string>('');
    const [searchType, setSearchType] = useState<string>('Profile');

    const resetSearchFilterKey = (key: keyof SearchFilter) => {
        setSearchFilter((previousFilter) => ({
            ...previousFilter,
            [key]: '',
        }));
    };

    function onSearchTypeSelect(_e, selection) {
        switch (selection) {
            case 'Profile':
                resetSearchFilterKey('clusterName');
                setClusterFilter('');
                break;
            case 'Cluster':
                resetSearchFilterKey('profileName');
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
                        onSearch={() => setSearchFilter({ ...searchFilter, scanName: scanFilter })}
                        onClear={() => {
                            setScanFilter('');
                            resetSearchFilterKey('scanName');
                        }}
                    />
                </ToolbarItem>
                <ToolbarItem variant="separator" />
                {searchType === 'Profile' ? (
                    <ProfileFilterInput
                        profileFilter={profileFilter}
                        setSearchFilter={setSearchFilter}
                        setProfileFilter={setProfileFilter}
                    />
                ) : (
                    <ClusterFilterInput
                        clusterFilter={clusterFilter}
                        setSearchFilter={setSearchFilter}
                        setClusterFilter={setClusterFilter}
                    />
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
                <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                    <Pagination
                        isCompact
                        itemCount={numberOfScanResults}
                        page={page}
                        perPage={perPage}
                        onSetPage={onSetPage}
                        onPerPageSelect={onPerPageSelect}
                    />
                </ToolbarItem>
            </ToolbarContent>
        </Toolbar>
    );
}

export default ScanResultsToolbar;
