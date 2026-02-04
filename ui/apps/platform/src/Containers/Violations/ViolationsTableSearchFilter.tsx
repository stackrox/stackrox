import { Toolbar, ToolbarContent, ToolbarGroup } from '@patternfly/react-core';

import type { SearchFilter } from 'types/search';
import useAnalytics from 'hooks/useAnalytics';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import type {
    CompoundSearchFilterConfig,
    OnSearchCallback,
} from 'Components/CompoundSearchFilter/types';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
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
        displayName: 'Cluster',
        searchCategory: 'ALERTS',
        attributes: [clusterIdAttribute, clusterLabelAttribute, clusterNameAttribute],
    },
    {
        displayName: 'Deployment',
        searchCategory: 'ALERTS',
        attributes: [
            DeploymentAnnotation,
            DeploymentID,
            DeploymentLabel,
            DeploymentName,
            DeploymentInactive, // Status
        ],
    },
    {
        displayName: 'Namespace',
        searchCategory: 'ALERTS',
        attributes: [NamespaceAnnotation, NamespaceID, NamespaceLabel, NamespaceName],
    },
    {
        displayName: 'Policy',
        searchCategory: 'ALERTS',
        attributes: [PolicyCategory, PolicyLifecycleStage, PolicyName, PolicySeverity],
    },
    {
        displayName: 'Policy violation',
        searchCategory: 'ALERTS',
        attributes: [AlertViolationTime, AlertEntityType], // non-alphabetical because no Name
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

    const onSearchHandler: OnSearchCallback = (payload) => {
        onSearch(payload);
        trackAppliedFilter('Policy Violations Filter Applied', payload);
    };

    return (
        <Toolbar>
            <ToolbarContent>
                <CompoundSearchFilter
                    config={searchFilterConfig}
                    defaultEntity="Policy"
                    searchFilter={searchFilter}
                    onSearch={onSearchHandler}
                    additionalContextFilter={additionalContextFilter}
                />
                <ToolbarGroup className="pf-v6-u-w-100">
                    <CompoundSearchFilterLabels
                        attributesSeparateFromConfig={[]}
                        config={searchFilterConfig}
                        onFilterChange={onFilterChange}
                        searchFilter={searchFilter}
                    />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default ViolationsTableSearchFilter;
