import React, { ReactElement } from 'react';
import {
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import {
    AdministrationEventLevel,
    getAdministrationEventsFilter,
    replaceSearchFilterDomain,
    replaceSearchFilterLevel,
    replaceSearchFilterResourceType,
} from 'services/AdministrationEventsService';
import { SearchFilter } from 'types/search';

import SearchFilterDomain from './SearchFilterDomain';
import SearchFilterLevel from './SearchFilterLevel';
import SearchFilterResourceType from './SearchFilterResourceType';
import UpdatedTimeOrUpdateButton from './UpdatedTimeOrUpdateButton';

export type AdministrationEventsToolbarProps = {
    count: number;
    countAvailable: number;
    isDisabled: boolean;
    lastUpdatedTime: string;
    page: number;
    perPage: number;
    setPage: (newPage: number) => void;
    setPerPage: (newPerPage: number) => void;
    searchFilter: SearchFilter;
    setSearchFilter: (newFilter: SearchFilter) => void;
    updateEvents: () => void;
};

function AdministrationEventsToolbar({
    count,
    countAvailable,
    isDisabled,
    lastUpdatedTime,
    page,
    perPage,
    setPage,
    setPerPage,
    searchFilter,
    setSearchFilter,
    updateEvents,
}: AdministrationEventsToolbarProps): ReactElement {
    function setDomain(domain: string | undefined) {
        setSearchFilter(replaceSearchFilterDomain(searchFilter, domain));
    }

    function setLevel(level: AdministrationEventLevel | undefined) {
        setSearchFilter(replaceSearchFilterLevel(searchFilter, level));
    }

    function setResourceType(resourceType: string | undefined) {
        setSearchFilter(replaceSearchFilterResourceType(searchFilter, resourceType));
    }

    const { domain, level, resourceType } = getAdministrationEventsFilter(searchFilter);

    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarGroup variant="filter-group">
                    <ToolbarItem>
                        <SearchFilterDomain
                            domain={domain && domain[0]}
                            isDisabled={isDisabled}
                            setDomain={setDomain}
                        />
                    </ToolbarItem>
                    <ToolbarItem>
                        <SearchFilterResourceType
                            isDisabled={isDisabled}
                            resourceType={resourceType && resourceType[0]}
                            setResourceType={setResourceType}
                        />
                    </ToolbarItem>
                    <ToolbarItem>
                        <SearchFilterLevel
                            isDisabled={isDisabled}
                            level={level && level[0]}
                            setLevel={setLevel}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup variant="button-group" alignment={{ default: 'alignRight' }}>
                    {lastUpdatedTime && (
                        <ToolbarItem>
                            <UpdatedTimeOrUpdateButton
                                countAvailable={countAvailable}
                                isAvailableEqualToPerPage={countAvailable === perPage}
                                isDisabled={isDisabled}
                                lastUpdatedTime={lastUpdatedTime}
                                updateEvents={updateEvents}
                            />
                        </ToolbarItem>
                    )}
                    <ToolbarItem variant="pagination">
                        <Pagination
                            isCompact
                            isDisabled={isDisabled}
                            itemCount={count}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                setPage(1);
                                setPerPage(newPerPage);
                            }}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default AdministrationEventsToolbar;
