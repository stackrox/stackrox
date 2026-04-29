import type { ReactElement } from 'react';
import { Alert, Flex, FormGroup } from '@patternfly/react-core';
import type { FormikProps } from 'formik';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import SearchFilterSelectInclusive from 'Components/CompoundSearchFilter/components/SearchFilterSelectInclusive';
import type {
    CompoundSearchFilterConfig,
    OnSearchPayload,
    SelectSearchFilterAttribute,
} from 'Components/CompoundSearchFilter/types';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import type { SearchFilter } from 'types/search';
import {
    getRequestQueryStringForSearchFilter,
    getSearchFilterFromSearchString,
} from 'utils/searchUtils';

export type FiltersQueryConfiguration = {
    vulnReportFilters: {
        query: string;
    };
};

export type FiltersQueryProps = {
    attributesSeparateFromConfig: SelectSearchFilterAttribute[];
    formik: FormikProps<FiltersQueryConfiguration>;
    searchFilterConfig: CompoundSearchFilterConfig;
};

function FiltersQuery({
    attributesSeparateFromConfig,
    formik,
    searchFilterConfig,
}: FiltersQueryProps): ReactElement {
    const searchFilter = getSearchFilterFromSearchString(formik.values.vulnReportFilters.query);

    function onFilterChange(searchFilterChanged: SearchFilter) {
        formik.setFieldValue(
            'vulnReportFilters.query',
            getRequestQueryStringForSearchFilter(searchFilterChanged)
        );
    }

    function onSearch(payload: OnSearchPayload) {
        onFilterChange(updateSearchFilter(searchFilter, payload));
    }

    return (
        <>
            {attributesSeparateFromConfig.map((attribute) => (
                <FormGroup key={attribute.searchTerm} label={attribute.displayName} fieldId="TODO">
                    <SearchFilterSelectInclusive
                        attribute={attribute}
                        onSearch={onSearch}
                        searchFilter={searchFilter}
                    />
                </FormGroup>
            ))}
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                <CompoundSearchFilter
                    config={searchFilterConfig}
                    onSearch={onSearch}
                    searchFilter={searchFilter}
                />
                {Object.keys(searchFilter).length !== 0 ? (
                    <CompoundSearchFilterLabels
                        attributesSeparateFromConfig={attributesSeparateFromConfig}
                        config={searchFilterConfig}
                        onFilterChange={onFilterChange}
                        searchFilter={searchFilter}
                    />
                ) : (
                    <Alert variant="warning" title="TODO" isInline component="p">
                        To be determined
                    </Alert>
                )}
            </Flex>
        </>
    );
}

export default FiltersQuery;
