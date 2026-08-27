import type { ReactElement } from 'react';
import {
    Flex,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
} from '@patternfly/react-core';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import SearchFilterSelectExclusiveSingle from 'Components/CompoundSearchFilter/components/SearchFilterSelectExclusiveSingle';
import SearchFilterSelectInclusive from 'Components/CompoundSearchFilter/components/SearchFilterSelectInclusive';
import type {
    CompoundSearchFilterConfig,
    OnSearchPayload,
    SelectSingleSearchFilterAttribute,
} from 'Components/CompoundSearchFilter/types';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import type { SearchFilter } from 'types/search';
import {
    applyRegexSearchModifiers,
    getRequestQueryStringForSearchFilter,
    getSearchFilterFromSearchString,
} from 'utils/searchUtils';

// Because filter property name differs for different report types,
// renderer is responsible to provide values from formik object.
export type FiltersQueryProps = {
    attributesSeparateFromConfig: SelectSingleSearchFilterAttribute[];
    error: string | undefined;
    query: string;
    searchFilterConfig: CompoundSearchFilterConfig;
    setQueryValue: (value: string, shouldValidate?: boolean) => void;
    touched: boolean | undefined;
};

function FiltersQuery({
    attributesSeparateFromConfig,
    error,
    query,
    searchFilterConfig,
    setQueryValue,
    touched,
}: FiltersQueryProps): ReactElement {
    const searchFilter = getSearchFilterFromSearchString(query);

    function onFilterChange(searchFilterChanged: SearchFilter) {
        setQueryValue(
            getRequestQueryStringForSearchFilter(applyRegexSearchModifiers(searchFilterChanged))
        );
    }

    function onSearch(payload: OnSearchPayload) {
        onFilterChange(updateSearchFilter(searchFilter, payload));
    }

    return (
        <>
            {attributesSeparateFromConfig.map((attribute) => (
                <FormGroup key={attribute.searchTerm} label={attribute.displayName} fieldId="TODO">
                    {attribute.inputType === 'select-exclusive-single' ? (
                        <SearchFilterSelectExclusiveSingle
                            attribute={attribute}
                            onSearch={onSearch}
                            searchFilter={searchFilter}
                        />
                    ) : (
                        <SearchFilterSelectInclusive
                            attribute={attribute}
                            onSearch={onSearch}
                            searchFilter={searchFilter}
                        />
                    )}
                </FormGroup>
            ))}
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                <CompoundSearchFilter
                    config={searchFilterConfig}
                    onSearch={onSearch}
                    searchFilter={searchFilter}
                />
                {Object.keys(searchFilter).length !== 0 ? (
                    <CompoundSearchFilterLabels
                        attributesSeparateFromConfig={attributesSeparateFromConfig}
                        config={searchFilterConfig}
                        hasClearFilters={false}
                        onFilterChange={onFilterChange}
                        searchFilter={searchFilter}
                    />
                ) : (
                    touched &&
                    error && (
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem variant="error">{error}</HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    )
                )}
            </Flex>
        </>
    );
}

export default FiltersQuery;
