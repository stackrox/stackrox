import { Toolbar, ToolbarContent, ToolbarGroup, ToolbarItem } from '@patternfly/react-core';

import type { SearchFilter } from 'types/search';
import useAnalytics from 'hooks/useAnalytics';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import type {
    CompoundSearchFilterConfig,
    OnSearchCallback,
} from 'Components/CompoundSearchFilter/types';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import SearchFilterChips, {
    makeFilterChipDescriptors,
} from 'Components/CompoundSearchFilter/components/SearchFilterChips';
import {
    Category as PolicyCategory,
    LifecycleStage as PolicyLifecycleStage,
    Name as PolicyName,
    Severity as PolicySeverity,
} from 'Components/CompoundSearchFilter/attributes/policy';
import {
    EntityType as AlertEntityType,
    ViolationTime as AlertViolationTime,
} from 'Components/CompoundSearchFilter/attributes/alert';
import {
    clusterIdAttribute,
    clusterLabelAttribute,
    clusterNameAttribute,
} from 'Components/CompoundSearchFilter/attributes/cluster';
import {
    Annotation as NamespaceAnnotation,
    ID as NamespaceID,
    Label as NamespaceLabel,
    Name as NamespaceName,
} from 'Components/CompoundSearchFilter/attributes/namespace';
import {
    Annotation as DeploymentAnnotation,
    ID as DeploymentID,
    Inactive as DeploymentInactive,
    Label as DeploymentLabel,
    Name as DeploymentName,
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

    const onSearchHandler: OnSearchCallback = (payload) => {
        onSearch(payload);
        trackAppliedFilter('Policy Violations Filter Applied', payload);
    };

    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarGroup className="pf-v6-u-w-100">
                    <ToolbarItem className="pf-v6-u-flex-1">
                        <CompoundSearchFilter
                            config={searchFilterConfig}
                            searchFilter={searchFilter}
                            onSearch={onSearchHandler}
                            additionalContextFilter={additionalContextFilter}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup className="pf-v6-u-w-100">
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
