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
} from 'Components/CompoundSearchFilter/attributes/policy';
import {
    ID as ClusterID,
    Name as ClusterName,
} from 'Components/CompoundSearchFilter/attributes/cluster';
import {
    ID as NamespaceID,
    Name as NamespaceName,
} from 'Components/CompoundSearchFilter/attributes/namespace';
import {
    ID as DeploymentID,
    Name as DeploymentName,
} from 'Components/CompoundSearchFilter/attributes/deployment';

const searchFilterConfig: CompoundSearchFilterConfig = [
    {
        displayName: 'Policy',
        searchCategory: 'ALERTS',
        attributes: [PolicyName, PolicyCategory, PolicyLifecycleStage],
    },
    {
        displayName: 'Cluster',
        searchCategory: 'ALERTS',
        attributes: [ClusterID, ClusterName],
    },
    {
        displayName: 'Namespace',
        searchCategory: 'ALERTS',
        attributes: [NamespaceID, NamespaceName],
    },
    {
        displayName: 'Deployment',
        searchCategory: 'ALERTS',
        attributes: [DeploymentID, DeploymentName],
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
