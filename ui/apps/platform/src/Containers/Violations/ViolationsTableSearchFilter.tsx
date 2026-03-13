import { Toolbar, ToolbarContent, ToolbarGroup, ToolbarItem } from '@patternfly/react-core';

import type { SearchFilter } from 'types/search';
import useAnalytics from 'hooks/useAnalytics';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import type { OnSearchCallback } from 'Components/CompoundSearchFilter/types';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import type { FilteredWorkflowView } from 'Components/FilteredWorkflowViewSelector/types';

import { getSearchFilterConfig } from './ViolationsTableSearchFilter.utils';

export type ViolationsTableSearchFilterProps = {
    searchFilter: SearchFilter;
    onFilterChange: (newFilter: SearchFilter) => void;
    onSearch: OnSearchCallback;
    additionalContextFilter: SearchFilter;
    filteredWorkflowView: FilteredWorkflowView;
};

function ViolationsTableSearchFilter({
    searchFilter,
    onFilterChange,
    onSearch,
    additionalContextFilter,
    filteredWorkflowView,
}: ViolationsTableSearchFilterProps) {
    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const searchFilterConfig = getSearchFilterConfig(filteredWorkflowView);

    const onSearchHandler: OnSearchCallback = (payload) => {
        onSearch(payload);
        trackAppliedFilter('Policy Violations Filter Applied', payload);
    };

    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarGroup className="pf-v5-u-w-100">
                    <ToolbarItem className="pf-v5-u-flex-1">
                        <CompoundSearchFilter
                            config={searchFilterConfig}
                            defaultEntity="Policy"
                            searchFilter={searchFilter}
                            onSearch={onSearchHandler}
                            additionalContextFilter={additionalContextFilter}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup className="pf-v5-u-w-100">
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
