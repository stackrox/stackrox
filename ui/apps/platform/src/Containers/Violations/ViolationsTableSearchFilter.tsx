import React from 'react';
import { Toolbar, ToolbarGroup, ToolbarContent, ToolbarItem } from '@patternfly/react-core';

import { SearchFilter } from 'types/search';
import { makeFilterChipDescriptors } from 'Components/CompoundSearchFilter/utils/utils';
import {
    CompoundSearchFilterConfig,
    OnSearchCallback,
} from 'Components/CompoundSearchFilter/types';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import {
    Category as PolicyCategory,
    Name as PolicyName,
    LifecycleStage as PolicyLifecycleStage,
    Severity as PolicySeverity,
    EnforcementAction as PolicyEnforcementAction,
} from 'Components/CompoundSearchFilter/attributes/policy';
import {
    ViolationTime as AlertViolationTime,
    EntityType as AlertEntityType,
} from 'Components/CompoundSearchFilter/attributes/alert';
import {
    Name as ClusterName,
    ID as ClusterID,
} from 'Components/CompoundSearchFilter/attributes/cluster';
import {
    ID as NamespaceID,
    Name as NamespaceName,
} from 'Components/CompoundSearchFilter/attributes/namespace';
import {
    ID as DeploymentID,
    Name as DeploymentName,
    Inactive as DeploymentInactive,
} from 'Components/CompoundSearchFilter/attributes/deployment';
import { Name as ResourceName } from 'Components/CompoundSearchFilter/attributes/resource';

const searchFilterConfig: CompoundSearchFilterConfig = [
    {
        displayName: 'Policy',
        searchCategory: 'ALERTS',
        attributes: [
            PolicyName,
            PolicyCategory,
            PolicySeverity,
            PolicyLifecycleStage,
            PolicyEnforcementAction,
        ],
    },
    {
        displayName: 'Policy violation',
        searchCategory: 'ALERTS',
        attributes: [AlertViolationTime, AlertEntityType],
    },
    {
        displayName: 'Cluster',
        searchCategory: 'ALERTS',
        attributes: [ClusterName, ClusterID],
    },
    {
        displayName: 'Namespace',
        searchCategory: 'ALERTS',
        attributes: [NamespaceName, NamespaceID],
    },
    {
        displayName: 'Deployment',
        searchCategory: 'ALERTS',
        attributes: [DeploymentName, DeploymentID, DeploymentInactive],
    },
    {
        displayName: 'Resource',
        searchCategory: 'ALERTS',
        attributes: [ResourceName],
    },
];

export type ViolationsTableSearchFilterProps = {
    searchFilter: SearchFilter;
    onSearch: OnSearchCallback;
};

function ViolationsTableSearchFilter({ searchFilter, onSearch }: ViolationsTableSearchFilterProps) {
    const filterChipGroupDescriptors = makeFilterChipDescriptors(searchFilterConfig);

    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarGroup className="pf-v5-u-w-100">
                    <ToolbarItem className="pf-v5-u-flex-1">
                        <CompoundSearchFilter
                            config={searchFilterConfig}
                            searchFilter={searchFilter}
                            onSearch={onSearch}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup className="pf-v5-u-w-100">
                    <SearchFilterChips filterChipGroupDescriptors={filterChipGroupDescriptors} />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default ViolationsTableSearchFilter;
