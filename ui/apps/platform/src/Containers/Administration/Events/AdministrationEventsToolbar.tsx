import type { ReactElement } from 'react';
import {
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import {
    getAdministrationEventsFilter,
    replaceSearchFilterCluster,
    replaceSearchFilterDeployment,
    replaceSearchFilterDomain,
    replaceSearchFilterLevel,
    replaceSearchFilterNamespace,
    replaceSearchFilterResourceType,
} from 'services/AdministrationEventsService';
import type { AdministrationEventLevel } from 'services/AdministrationEventsService';
import type { SearchFilter } from 'types/search';

import SearchFilterCluster from './SearchFilterCluster';
import SearchFilterDeployment from './SearchFilterDeployment';
import SearchFilterDomain from './SearchFilterDomain';
import SearchFilterLevel from './SearchFilterLevel';
import SearchFilterNamespace from './SearchFilterNamespace';
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
    function setCluster(cluster: string | undefined) {
        setSearchFilter(replaceSearchFilterCluster(searchFilter, cluster));
    }

    function setDeploymentFilter(deployment: string | undefined) {
        setSearchFilter(replaceSearchFilterDeployment(searchFilter, deployment));
    }

    function setDomain(domain: string | undefined) {
        setSearchFilter(replaceSearchFilterDomain(searchFilter, domain));
    }

    function setLevel(level: AdministrationEventLevel | undefined) {
        setSearchFilter(replaceSearchFilterLevel(searchFilter, level));
    }

    function setNamespace(namespace: string | undefined) {
        setSearchFilter(replaceSearchFilterNamespace(searchFilter, namespace));
    }

    function setResourceType(resourceType: string | undefined) {
        setSearchFilter(replaceSearchFilterResourceType(searchFilter, resourceType));
    }

    const { cluster, deployment, domain, level, namespace, resourceType } =
        getAdministrationEventsFilter(searchFilter);

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
                </ToolbarGroup>
                <ToolbarGroup variant="filter-group">
                    <ToolbarItem>
                        <SearchFilterResourceType
                            isDisabled={isDisabled}
                            resourceType={resourceType && resourceType[0]}
                            setResourceType={setResourceType}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup variant="filter-group">
                    <ToolbarItem>
                        <SearchFilterLevel
                            isDisabled={isDisabled}
                            level={level && level[0]}
                            setLevel={setLevel}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup variant="filter-group">
                    <ToolbarItem>
                        <SearchFilterCluster
                            cluster={cluster && cluster[0]}
                            isDisabled={isDisabled}
                            setCluster={setCluster}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup variant="filter-group">
                    <ToolbarItem>
                        <SearchFilterNamespace
                            isDisabled={isDisabled}
                            namespace={namespace && namespace[0]}
                            setNamespace={setNamespace}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup variant="filter-group">
                    <ToolbarItem>
                        <SearchFilterDeployment
                            deployment={deployment && deployment[0]}
                            isDisabled={isDisabled}
                            setDeployment={setDeploymentFilter}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup variant="action-group" align={{ default: 'alignEnd' }}>
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
