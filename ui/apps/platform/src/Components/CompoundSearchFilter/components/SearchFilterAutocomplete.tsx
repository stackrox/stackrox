import React, { useMemo, useRef, useState } from 'react';
import {
    Select,
    SelectOption,
    SelectList,
    SelectOptionProps,
    MenuToggle,
    MenuToggleElement,
    TextInputGroup,
    TextInputGroupMain,
    TextInputGroupUtilities,
    Button,
    Skeleton,
    Flex,
    debounce,
} from '@patternfly/react-core';
import { ArrowRightIcon, SearchIcon, TimesIcon } from '@patternfly/react-icons';
import { useQuery } from '@apollo/client';
import SEARCH_AUTOCOMPLETE_QUERY, {
    SearchAutocompleteQueryResponse,
} from 'queries/searchAutocomplete';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { SearchFilter } from 'types/search';
import { ensureString } from 'utils/ensure';

type SearchFilterAutocompleteProps = {
    searchCategory: string;
    searchTerm: string;
    value: string;
    onChange: (value: string) => void;
    onSearch: (value: string) => void;
    textLabel: string;
    searchFilter: SearchFilter;
    additionalContextFilter?: SearchFilter;
};

function getSelectOptions(
    data: SearchAutocompleteQueryResponse | undefined,
    isLoading: boolean,
    filterValue: string
): SelectOptionProps[] {
    if (isLoading) {
        return [
            {
                isDisabled: true,
                value: 'autocomplete-options-loading',
                children: (
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                    >
                        <Skeleton screenreaderText="Loading suggested options" width="100%" />
                    </Flex>
                ),
            },
        ];
    }

    if (data && data.searchAutocomplete && data.searchAutocomplete.length !== 0) {
        const options: SelectOptionProps[] = data.searchAutocomplete.map((optionValue) => {
            return {
                value: optionValue,
                children: optionValue,
            };
        });
        return options;
    }

    if (filterValue === '') {
        return [
            {
                isDisabled: true,
                value: filterValue,
                children: 'No options',
            },
        ];
    }

    return [
        {
            value: filterValue,
            children: `Add "${filterValue}"`,
        },
    ];
}

function SearchFilterAutocomplete({
    searchCategory,
    searchTerm,
    value,
    onChange,
    onSearch,
    textLabel,
    searchFilter,
    additionalContextFilter,
}: SearchFilterAutocompleteProps) {
    const [isOpen, setIsOpen] = useState(false);
    const [filterValue, setFilterValue] = useState('');
    const [isTyping, setIsTyping] = useState(false);
    const [focusedItemIndex, setFocusedItemIndex] = useState<number | null>(null);
    const [activeItem, setActiveItem] = useState<string | null>(null);
    const textInputRef = useRef<HTMLInputElement>();

    const setFilterValueDebounced = useMemo(
        () =>
            debounce((newValue: string) => {
                setFilterValue(newValue);
                setActiveItem(null);
                setFocusedItemIndex(null);
                setIsTyping(false);
            }, 500),
        []
    );

    const autocompleteSearchString = `${searchTerm}:${filterValue ? `r/${filterValue}` : ''}`;

    const searchContext = {
        ...searchFilter,
        ...additionalContextFilter,
    };
    const filteredSearchContext = Object.keys(searchContext).reduce((acc, key) => {
        // Autocomplete requests for some filters never return results if there is a 'Fixable' search filter
        // included in the query.
        // We also need to exclude the current search term from the autocomplete query otherwise following
        // autocomplete requests will return only the current search term as a suggestion.
        if (key !== 'FIXABLE' && key.toLowerCase() !== searchTerm.toLowerCase()) {
            acc[key] = searchContext[key];
        }
        return acc;
    }, {});
    const autocompleteContextString = getRequestQueryStringForSearchFilter(filteredSearchContext);

    const autocompleteQuery =
        autocompleteContextString !== ''
            ? [autocompleteContextString, autocompleteSearchString].join('+')
            : autocompleteSearchString;

    const { data: rawData, loading: isLoading } = useQuery<SearchAutocompleteQueryResponse>(
        SEARCH_AUTOCOMPLETE_QUERY,
        {
            variables: {
                query: autocompleteQuery,
                categories: searchCategory,
            },
        }
    );
    // Filter out empty strings
    const data: SearchAutocompleteQueryResponse = {
        searchAutocomplete: rawData?.searchAutocomplete?.filter((item) => item !== '').sort() ?? [],
    };

    const selectOptions: SelectOptionProps[] = getSelectOptions(
        data,
        isLoading || isTyping,
        filterValue
    );

    const onToggleClick = () => {
        setIsOpen(!isOpen);
    };

    const onSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        if (value) {
            onChange(ensureString(value));
            setFilterValue('');
        }
        setIsOpen(false);
        setFocusedItemIndex(null);
        setActiveItem(null);
    };

    const onTextInputChange = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
        onChange(value);
        if (!isOpen) {
            setIsOpen(true);
        } else if (isOpen && value === '') {
            setIsOpen(false);
        }
        setFilterValueDebounced(value);
        setIsTyping(true);
    };

    const handleMenuArrowKeys = (key: 'ArrowUp' | 'ArrowDown') => {
        let indexToFocus;

        if (isOpen) {
            if (key === 'ArrowUp') {
                // When no index is set or at the first index, focus to the last, otherwise decrement focus index
                if (focusedItemIndex === null || focusedItemIndex === 0) {
                    indexToFocus = selectOptions.length - 1;
                } else {
                    indexToFocus = focusedItemIndex - 1;
                }
            }

            if (key === 'ArrowDown') {
                // When no index is set or at the last index, focus to the first, otherwise increment focus index
                if (focusedItemIndex === null || focusedItemIndex === selectOptions.length - 1) {
                    indexToFocus = 0;
                } else {
                    indexToFocus = focusedItemIndex + 1;
                }
            }

            setFocusedItemIndex(indexToFocus);
            const focusedItem = selectOptions.filter((option) => !option.isDisabled)[indexToFocus];
            setActiveItem(`select-typeahead-${focusedItem?.value?.replace(' ', '-space-')}`);
        }
    };

    const onInputKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
        const enabledMenuItems = selectOptions.filter((option) => !option.isDisabled);
        const [firstMenuItem] = enabledMenuItems;
        const focusedItem = focusedItemIndex ? enabledMenuItems[focusedItemIndex] : firstMenuItem;

        switch (event.key) {
            // Select the first available option
            case 'Enter':
                if (isOpen) {
                    const newValue = ensureString(focusedItem.value);
                    onChange(newValue);
                    onSearch(newValue);
                    setFilterValue('');
                }

                setIsOpen(false);
                setFocusedItemIndex(null);
                setActiveItem(null);

                break;
            case 'Tab':
            case 'Escape':
                setIsOpen(false);
                setActiveItem(null);
                break;
            case 'ArrowUp':
            case 'ArrowDown':
                event.preventDefault();
                handleMenuArrowKeys(event.key);
                break;
            default:
        }
    };

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            variant="typeahead"
            onClick={onToggleClick}
            isExpanded={isOpen}
            isFullWidth
            aria-labelledby="Filter results menu toggle"
        >
            <TextInputGroup isPlain>
                <TextInputGroupMain
                    value={value}
                    onClick={onToggleClick}
                    onChange={onTextInputChange}
                    onKeyDown={onInputKeyDown}
                    autoComplete="off"
                    innerRef={textInputRef}
                    placeholder={textLabel}
                    {...(activeItem && { 'aria-activedescendant': activeItem })}
                    role="combobox"
                    isExpanded={isOpen}
                    aria-controls="select-typeahead-listbox"
                    aria-label={textLabel}
                    icon={<SearchIcon />}
                />
                <TextInputGroupUtilities>
                    {!!value && (
                        <Button
                            variant="plain"
                            onClick={() => {
                                onChange('');
                                setFilterValue('');
                                textInputRef?.current?.focus();
                            }}
                            aria-label="Clear input value"
                        >
                            <TimesIcon aria-hidden />
                        </Button>
                    )}
                </TextInputGroupUtilities>
            </TextInputGroup>
        </MenuToggle>
    );

    return (
        <>
            <Select
                isOpen={isOpen}
                selected={value}
                onSelect={onSelect}
                onOpenChange={() => {
                    setIsOpen(false);
                }}
                toggle={toggle}
            >
                <SelectList id="select-typeahead-listbox" aria-label="Filter results select menu">
                    {selectOptions.map((option, index) => (
                        <SelectOption
                            key={option.value || option.children}
                            isFocused={focusedItemIndex === index}
                            isSelected={false}
                            className={option.className}
                            onClick={() => {
                                onChange(option.value);
                                onSearch(option.value);
                            }}
                            id={`select-typeahead-${option?.value?.replace(' ', '-space-')}`}
                            {...option}
                            ref={null}
                        />
                    ))}
                </SelectList>
            </Select>
            <Button
                variant="control"
                aria-label="Apply autocomplete input to search"
                onClick={() => {
                    onSearch(value);
                }}
            >
                <ArrowRightIcon />
            </Button>
        </>
    );
}

export default SearchFilterAutocomplete;
