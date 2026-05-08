import { useState } from 'react';
import type { ReactElement } from 'react';
import { Flex } from '@patternfly/react-core';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import type {
    CompoundSearchFilterConfig,
    OnSearchPayload,
} from 'Components/CompoundSearchFilter/types';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import type { EntityScope, EntityScopeRule } from 'services/ReportsService.types';
import type { SearchFilter } from 'types/search';

// Why searchFilterConfig and get functions as props?
// Because nodes have cluster but not namespace nor deployment as entities.
// Why not formik object as prop like FiltersQuery?
// To allow possible reuse for saved entity scope in addition to report configuration.
export type EntityScopeCompoundSearchFilterProps = {
    entityScope: EntityScope | null;
    getEntityScopeRulesFromSearchFilter: (searchFilter: SearchFilter) => EntityScopeRule[];
    getSearchFilterFromEntityScopeRules: (rules: EntityScopeRule[]) => SearchFilter;
    searchFilterConfig: CompoundSearchFilterConfig;
    setEntityScope: (entityScope: EntityScope) => void;
};

// Although entity scope has structured rules instead of query string,
// interaction via compound search filter is similar for resources and filters.
// Caller is responsible for conditional rendering if no rules.
// For example, rules are expected for Deployed imaged but not necessarily for Watched images.
function EntityScopeCompoundSearchFilter({
    entityScope,
    getEntityScopeRulesFromSearchFilter,
    getSearchFilterFromEntityScopeRules,
    searchFilterConfig,
    setEntityScope,
}: EntityScopeCompoundSearchFilterProps): ReactElement {
    const [searchFilter, setSearchFilter] = useState<SearchFilter>(
        entityScope ? getSearchFilterFromEntityScopeRules(entityScope.rules) : {}
    );

    function onFilterChange(searchFilterChanged: SearchFilter) {
        const rules = getEntityScopeRulesFromSearchFilter(searchFilterChanged);
        setEntityScope({ rules });
        setSearchFilter(searchFilterChanged);
    }

    function onSearch(payload: OnSearchPayload) {
        onFilterChange(updateSearchFilter(searchFilter, payload));
    }

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
            <CompoundSearchFilter
                config={searchFilterConfig}
                isDisabled={entityScope === null}
                onSearch={onSearch}
                searchFilter={searchFilter}
            />
            {entityScope && (
                <CompoundSearchFilterLabels
                    attributesSeparateFromConfig={[]}
                    config={searchFilterConfig}
                    hasClearFilters={false}
                    onFilterChange={onFilterChange}
                    searchFilter={searchFilter}
                />
            )}
        </Flex>
    );
}

export default EntityScopeCompoundSearchFilter;
