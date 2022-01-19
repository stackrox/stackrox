import { useQuery } from '@apollo/client';

import { searchCategories } from 'constants/entityTypes';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';

type SearchOptionsResponse = {
    data?: {
        searchOptions?: string[];
    };
};

/*
 * This hook uses the Apollo client to retrieve search options for the given category
 */
function useSearchOptions(searchCategory: string): string[] {
    const searchQueryOptions = {
        variables: {
            categories: [searchCategories[searchCategory]],
        },
    };

    const { data }: SearchOptionsResponse = useQuery(SEARCH_OPTIONS_QUERY, searchQueryOptions);
    const searchOptions = data?.searchOptions || [];

    return searchOptions;
}

export default useSearchOptions;
