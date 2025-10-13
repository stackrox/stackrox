import React from 'react';
import { Toolbar, ToolbarGroup, ToolbarContent, ToolbarItem } from '@patternfly/react-core';

import type { SearchFilter } from 'types/search';
import useAnalytics from 'hooks/useAnalytics';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import { makeFilterChipDescriptors } from 'Components/CompoundSearchFilter/utils/utils';
import type {
    CompoundSearchFilterConfig,
    OnSearchCallback,
    OnSearchPayload,
} from 'Components/CompoundSearchFilter/types';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import {
    Category as PolicyCategory,
    Name as PolicyName,
    LifecycleStage as PolicyLifecycleStage,
    Severity as PolicySeverity,
} from 'Components/CompoundSearchFilter/attributes/policy';
import {
    ViolationTime as AlertViolationTime,
    EntityType as AlertEntityType,
} from 'Components/CompoundSearchFilter/attributes/alert';
import {
    clusterNameAttribute,
    clusterIdAttribute,
    clusterLabelAttribute,
} from 'Components/CompoundSearchFilter/attributes/cluster';
import {
    ID as NamespaceID,
    Name as NamespaceName,
    Label as NamespaceLabel,
    Annotation as NamespaceAnnotation,
} from 'Components/CompoundSearchFilter/attributes/namespace';
import {
    ID as DeploymentID,
    Name as DeploymentName,
    Inactive as DeploymentInactive,
    Label as DeploymentLabel,
    Annotation as DeploymentAnnotation,
} from 'Components/CompoundSearchFilter/attributes/deployment';
import { Name as ResourceName } from 'Components/CompoundSearchFilter/attributes/resource';

const searchFilterConfig: CompoundSearchFilterConfig = [
    {
        displayName: 'Policy',
        searchCategory: 'ALERTS',
        attributes: [PolicyName, PolicyCategory, PolicySeverity, PolicyLifecycleStage],
    },
    {
        displayName: 'Policy violation',
        searchCategory: 'ALERTS',
        attributes: [AlertViolationTime, AlertEntityType],
    },
    {
        displayName: 'Cluster',
        searchCategory: 'ALERTS',
        attributes: [clusterNameAttribute, clusterIdAttribute, clusterLabelAttribute],
    },
    {
        displayName: 'Namespace',
        searchCategory: 'ALERTS',
        attributes: [NamespaceName, NamespaceID, NamespaceLabel, NamespaceAnnotation],
    },
    {
        displayName: 'Deployment',
        searchCategory: 'ALERTS',
        attributes: [
            DeploymentName,
            DeploymentID,
            DeploymentLabel,
            DeploymentAnnotation,
            DeploymentInactive,
        ],
    },
    {
        displayName: 'Resource',
        searchCategory: 'ALERTS',
        attributes: [ResourceName],
    },
];

export type ViolationsTableSearchFilterProps = {
    searchFilter: SearchFilter;
    onFilterChange: (newFilter: SearchFilter) => void;
    onSearch: OnSearchCallback;
    additionalContextFilter: SearchFilter;
};

function ViolationsTableSearchFilter({
    searchFilter,
    onFilterChange,
    onSearch,
    additionalContextFilter,
}: ViolationsTableSearchFilterProps) {
    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const filterChipGroupDescriptors = makeFilterChipDescriptors(searchFilterConfig);

    function onSearchHandler(payload: OnSearchPayload) {
        onSearch(payload);
        trackAppliedFilter('Policy Violations Filter Applied', payload);
    }

    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarGroup className="pf-v5-u-w-100">
                    <ToolbarItem className="pf-v5-u-flex-1">
                        <CompoundSearchFilter
                            config={searchFilterConfig}
                            searchFilter={searchFilter}
                            onSearch={onSearchHandler}
                            additionalContextFilter={additionalContextFilter}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup className="pf-v5-u-w-100">
                    <SearchFilterChips
                        searchFilter={searchFilter}
                        onFilterChange={onFilterChange}
                        filterChipGroupDescriptors={filterChipGroupDescriptors}
                    />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default ViolationsTableSearchFilter;
