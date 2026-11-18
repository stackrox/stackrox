import { Toolbar, ToolbarContent, ToolbarGroup } from '@patternfly/react-core';

import type { SearchFilter } from 'types/search';
import useFeatureFlags from 'hooks/useFeatureFlags';
import type { OnSearchCallback } from 'Components/CompoundSearchFilter/types';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import { getSearchFilterConfigWithFeatureFlagDependency } from 'Components/CompoundSearchFilter/utils/utils';

import { getLegacySearchFilterAttributes, searchFilterConfig } from './riskSearchFilterConfig';

export type RiskTableSearchFilterProps = {
    searchFilter: SearchFilter;
    onSearch: OnSearchCallback;
    onFilterChange: (newFilter: SearchFilter) => void;
    // Scopes value autocomplete to the active workflow view (Applications/Platform/Full).
    additionalContextFilter: SearchFilter;
};

function RiskTableSearchFilter({
    searchFilter,
    onSearch,
    onFilterChange,
    additionalContextFilter,
}: RiskTableSearchFilterProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const config = getSearchFilterConfigWithFeatureFlagDependency(
        isFeatureFlagEnabled,
        searchFilterConfig
    );

    return (
        <Toolbar>
            <ToolbarContent>
                <CompoundSearchFilter
                    config={config}
                    defaultEntity="Deployment"
                    searchFilter={searchFilter}
                    additionalContextFilter={additionalContextFilter}
                    onSearch={onSearch}
                />
                <ToolbarGroup className="pf-v6-u-w-100">
                    <CompoundSearchFilterLabels
                        attributesSeparateFromConfig={getLegacySearchFilterAttributes(
                            searchFilter,
                            config
                        )}
                        config={config}
                        onFilterChange={onFilterChange}
                        searchFilter={searchFilter}
                    />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default RiskTableSearchFilter;
