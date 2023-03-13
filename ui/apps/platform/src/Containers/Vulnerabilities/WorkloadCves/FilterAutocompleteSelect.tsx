import React, { useCallback, useState } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';
import { useQuery } from '@apollo/client';

import { SearchFilter } from 'types/search';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import { searchCategories } from 'constants/entityTypes';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import FilterResourceDropdown, { Resource } from './FilterResourceDropdown';

function getOptions(data: string[] | undefined): React.ReactElement[] | undefined {
    return data?.map((value) => <SelectOption key={value} value={value} />);
}

type FilterAutocompleteSelectProps = {
    searchFilter: SearchFilter;
    setSearchFilter: (s) => void;
    resourceContext?: Resource;
};

function FilterAutocompleteSelect({
    searchFilter,
    setSearchFilter,
    resourceContext,
}: FilterAutocompleteSelectProps) {
    const [resource, setResource] = useState<Resource>('DEPLOYMENT');
    const { isOpen, onToggle } = useSelectToggle();
    // const [typeahead, setTypeahead] = useState(selectedOption);
    // const [autoCompleteQuery, setAutoCompleteQuery] = useState('');
    const variables = {
        query: getRequestQueryStringForSearchFilter({ [resource]: searchFilter[resource] }),
        categories: searchCategories[resource],
    };

    console.log(variables);
    const { loading, data, error } = useQuery(SEARCH_AUTOCOMPLETE_QUERY, { variables });

    // When isTyping is true, autocomplete results will not be displayed. This prevents
    // a clunky UX where the dropdown results and the user text get out of sync.
    const [isTyping, setIsTyping] = useState(false);
    console.log(data);

    function onSelect(newValue) {
        setSearchFilter({
            ...searchFilter,
            [resource]: newValue,
        });
    }

    // const autocompleteCallback = useCallback(() => {
    //     const shouldMakeRequest = isOpen;
    //     if (shouldMakeRequest) {
    //         const req = generateRequest(collection);
    //         const { request, cancel } = getCollectionAutoComplete(
    //             req.resourceSelectors,
    //             autocompleteField,
    //             typeahead
    //         );
    //         request.finally(() => setIsTyping(false));
    //         return { request, cancel };
    //     }
    //     return {
    //         request: new Promise<string[]>((resolve) => {
    //             setIsTyping(false);
    //             resolve([]);
    //         }),
    //         cancel: () => {},
    //     };
    // }, [autocompleteField, collection, isOpen, typeahead]);

    return (
        <>
            <FilterResourceDropdown
                setResource={setResource}
                resource={resource}
                resourceContext={resourceContext}
            />
            <Select
                aria-label={`Filter by ${resource as string}`}
                onSelect={(e, value) => {
                    onSelect(value);
                }}
                onToggle={onToggle}
                isOpen={isOpen}
                placeholder={`Filter by ${resource as string}`}
                variant="typeahead"
                isCreatable
                createText="Add"
                selections={searchFilter[resource]}
            >
                {getOptions(isTyping ? [] : data)}
            </Select>
        </>
    );
}

export default FilterAutocompleteSelect;
