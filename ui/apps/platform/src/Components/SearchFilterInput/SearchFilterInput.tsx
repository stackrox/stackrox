import React, { useEffect, useState } from 'react';

import SearchInput, { createSearchModifiers } from 'Components/SearchInput';
import { SearchCategory, fetchAutoCompleteResults } from 'services/SearchService';
import { SearchEntry, SearchFilter } from 'types/search';

type SearchFilterInputProps = {
    className: string;
    handleChangeSearchFilter: (searchFilter: SearchFilter) => void;
    placeholder: string;
    searchCategory?: SearchCategory;
    searchFilter: SearchFilter;
    searchOptions: string[]; // differs from searchOptions prop of SearchInput
    autocompleteQueryPrefix?: string;
    isDisabled?: boolean;
};

/*
 * Render SearchInput element for searchFilter with searchCategory and searchOptions.
 * Initial value of searchOptions might be empty array until response from request.
 *
 * Encapsulate autoComplete request for values of an option for a searchCategory.
 *
 * Replacement for ReduxSearchInput or URLSearchInput:
 * Caller parses searchFilter from search query string in URL.
 * Caller updates URL whenever handleChangeSearchFilter is called.
 */
function SearchFilterInput({
    className,
    handleChangeSearchFilter,
    placeholder,
    searchCategory,
    searchFilter,
    searchOptions,
    autocompleteQueryPrefix,
    isDisabled = false,
}: SearchFilterInputProps) {
    const [autoCompleteValues, setAutoCompleteValues] = useState<string[]>([]);
    const [valuelessOption, setValuelessOption] = useState('');

    useEffect(() => {
        if (valuelessOption.length !== 0) {
            const query = autocompleteQueryPrefix
                ? `${autocompleteQueryPrefix}+${getEntryValueForOption(valuelessOption)}`
                : getEntryValueForOption(valuelessOption);

            fetchAutoCompleteResults({
                categories: searchCategory && [searchCategory],
                query,
            })
                .then((values) => {
                    setAutoCompleteValues(values);
                })
                .catch(() => {});
        }
    }, [searchCategory, autocompleteQueryPrefix, valuelessOption]);

    function handleChangeSearchEntries(searchEntries: SearchEntry[]) {
        setValuelessOption(getValuelessOption(searchEntries));
        handleChangeSearchFilter(getSearchFilterForEntries(searchEntries));
    }

    /*
     * Until response from request for searchOptions:
     * disable SearchInput;
     * render empty array of search entries.
     * Assume that the page waits to make its main request, unless searchFilter is empty.
     *
     * Although SearchFilterInput might not be global, without isGlobal prop:
     * SearchInput componentWillUnmount calls setSearchOptions([])
     * which causes this component to call handleChangeSearchFilter,
     * which causes the page to update its URL and override a link to another page.
     */
    return (
        <SearchInput
            autoCompleteResults={autoCompleteValues}
            className={className}
            isDisabled={searchOptions.length === 0 || isDisabled}
            isGlobal
            placeholder={placeholder}
            searchModifiers={createSearchModifiers(searchOptions)}
            searchOptions={getSearchEntriesForFilter(searchFilter, searchOptions)}
            setSearchOptions={handleChangeSearchEntries}
        />
    );
}

/*
 * Return search entry array for search filter object.
 *
 * Because search filter object might have been parsed from search query string of URL,
 * filter to include only properties whose key is in search options.
 * Therefore, the returned array is empty before response to search options request.
 */
function getSearchEntriesForFilter(
    searchFilter: SearchFilter,
    searchOptions: string[]
): SearchEntry[] {
    const searchEntries: SearchEntry[] = [];

    Object.entries(searchFilter).forEach(([key, valueOrValues]) => {
        if (searchOptions.includes(key)) {
            const valueForOption = getEntryValueForOption(key);
            searchEntries.push({
                type: 'categoryOption',
                label: valueForOption,
                value: valueForOption,
            });

            if (Array.isArray(valueOrValues)) {
                valueOrValues.forEach((value) => {
                    searchEntries.push({
                        label: value,
                        value,
                    });
                });
            } else if (valueOrValues && valueOrValues.length !== 0) {
                searchEntries.push({
                    label: valueOrValues,
                    value: valueOrValues,
                });
            }
        }
    });

    return searchEntries;
}

/*
 * Return search filter object for changed search entries.
 *
 * Assume search entries have been filtered to include only categoryOption in search options.
 */
function getSearchFilterForEntries(searchEntries: SearchEntry[]): SearchFilter {
    const searchFilter: SearchFilter = {};

    let i = 0;
    while (i < searchEntries.length) {
        if (searchEntries[i].type === 'categoryOption') {
            const key = getOptionForEntryValue(searchEntries[i].value);
            const values: string[] = [];

            i += 1;
            while (i < searchEntries.length && searchEntries[i].type !== 'categoryOption') {
                values.push(searchEntries[i].value);
                i += 1;
            }

            switch (values.length) {
                case 0:
                    searchFilter[key] = ''; // valueless option
                    break;
                case 1:
                    searchFilter[key] = values[0]; // eslint-disable-line prefer-destructuring
                    break;
                default:
                    searchFilter[key] = values;
            }
        } else {
            i += 1;
        }
    }

    return searchFilter;
}

/*
 * Return the last search option if it does not have a value; otherwise empty string.
 */
function getValuelessOption(searchEntries: SearchEntry[]): string {
    if (searchEntries.length !== 0) {
        const { type, value } = searchEntries[searchEntries.length - 1];
        if (type === 'categoryOption') {
            return getOptionForEntryValue(value);
        }
    }

    return '';
}

function getEntryValueForOption(option: string): string {
    return `${option}:`;
}

function getOptionForEntryValue(label: string): string {
    return label.replace(':', '');
}

export default SearchFilterInput;
