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
import TimesIcon from '@patternfly/react-icons/dist/esm/icons/times-icon';
import { ArrowRightIcon } from '@patternfly/react-icons';
import { useQuery } from '@apollo/client';
import SEARCH_AUTOCOMPLETE_QUERY, {
    SearchAutocompleteQueryResponse,
} from 'queries/searchAutocomplete';

type SearchFilterAutocompleteProps = {
    searchCategory: string;
    searchTerm: string;
    value: string;
    onChange: (value: string) => void;
    onSearch: (value: string) => void;
    textLabel: string;
};

function getSelectOptions(
    data: SearchAutocompleteQueryResponse | undefined,
    isLoading: boolean
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
                        <Skeleton screenreaderText="Loading suggested options 1" width="100%" />
                        <Skeleton screenreaderText="Loading suggested options 2" width="75%" />
                        <Skeleton screenreaderText="Loading suggested options 3" width="90%" />
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

    return [
        {
            isDisabled: false,
            children: `No results found`,
            value: 'no results',
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
                setIsTyping(false);
            }, 500),
        []
    );

    const { data, loading: isLoading } = useQuery<SearchAutocompleteQueryResponse>(
        SEARCH_AUTOCOMPLETE_QUERY,
        {
            variables: {
                query: `${searchTerm}:${filterValue ? `r/${filterValue}` : ''}`,
                categories: searchCategory,
            },
        }
    );

    const selectOptions: SelectOptionProps[] = getSelectOptions(data, isLoading || isTyping);

    React.useEffect(() => {
        if (filterValue && !isOpen) {
            // Open the menu when the input value changes and the new value is not empty
            setIsOpen(true);
        }
        setActiveItem(null);
        setFocusedItemIndex(null);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [filterValue]);

    const onToggleClick = () => {
        setIsOpen(!isOpen);
    };

    const onSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        if (value && value !== 'no results') {
            onChange(value as string);
            setFilterValue('');
        }
        setIsOpen(false);
        setFocusedItemIndex(null);
        setActiveItem(null);
    };

    const onTextInputChange = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
        onChange(value);
        setFilterValueDebounced(value);
        setIsTyping(true);
    };

    const handleMenuArrowKeys = (key: string) => {
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
            setActiveItem(`select-typeahead-${focusedItem.value.replace(' ', '-')}`);
        }
    };

    const onInputKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
        const enabledMenuItems = selectOptions.filter((option) => !option.isDisabled);
        const [firstMenuItem] = enabledMenuItems;
        const focusedItem = focusedItemIndex ? enabledMenuItems[focusedItemIndex] : firstMenuItem;

        switch (event.key) {
            // Select the first available option
            case 'Enter':
                if (isOpen && focusedItem.value !== 'no results') {
                    onChange(String(focusedItem.children));
                    setFilterValue('');
                }

                setIsOpen((prevIsOpen) => !prevIsOpen);
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
            aria-labelledby={`${textLabel} menu toggle`}
        >
            <TextInputGroup isPlain>
                <TextInputGroupMain
                    value={value}
                    onClick={onToggleClick}
                    onChange={onTextInputChange}
                    onKeyDown={onInputKeyDown}
                    id="typeahead-select-input"
                    autoComplete="off"
                    innerRef={textInputRef}
                    placeholder={textLabel}
                    {...(activeItem && { 'aria-activedescendant': activeItem })}
                    role="combobox"
                    isExpanded={isOpen}
                    aria-controls="select-typeahead-listbox"
                    aria-label={textLabel}
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
                aria-label={`${textLabel} select menu`}
                isOpen={isOpen}
                selected={value}
                onSelect={onSelect}
                onOpenChange={() => {
                    setIsOpen(false);
                }}
                toggle={toggle}
            >
                <SelectList id="select-typeahead-listbox">
                    {selectOptions.map((option, index) => (
                        <SelectOption
                            key={option.value || option.children}
                            isFocused={focusedItemIndex === index}
                            className={option.className}
                            onClick={() => onChange(option.value)}
                            id={`select-typeahead-${option.value.replace(' ', '-')}`}
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
