import React, { ReactElement, useState, useMemo } from 'react';
import {
    Select,
    SelectOption,
    MenuToggle,
    MenuToggleElement,
    SelectList,
    TextInputGroup,
    TextInputGroupMain,
    TextInputGroupUtilities,
    Button,
} from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';

export type TypeaheadCheckboxSelectOption = {
    value: string;
    label?: string;
    disabled?: boolean;
};

export type TypeaheadCheckboxSelectProps = {
    id: string;
    menuToggleId?: string;
    selections: string[];
    onChange: (selections: string[]) => void;
    options: TypeaheadCheckboxSelectOption[];
    placeholder?: string;
    isDisabled?: boolean;
    toggleAriaLabel?: string;
    onBlur?: React.FocusEventHandler<HTMLDivElement>;
    menuAppendTo?: () => HTMLElement;
    maxHeight?: string;
    direction?: 'up' | 'down';
};

function TypeaheadCheckboxSelect({
    id,
    menuToggleId,
    selections,
    onChange,
    options,
    placeholder = 'Type to search...',
    isDisabled = false,
    toggleAriaLabel,
    onBlur,
    menuAppendTo,
    maxHeight = '300px',
    direction = 'down',
}: TypeaheadCheckboxSelectProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);
    const [inputValue, setInputValue] = useState('');
    const [focusedItemIndex, setFocusedItemIndex] = useState<number>(-1);

    function onSelect(
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        selection: string | number | undefined
    ) {
        if (typeof selection === 'string') {
            if (selections.includes(selection)) {
                onChange(selections.filter((item) => item !== selection));
            } else {
                onChange([...selections, selection]);
            }
            // Keep the dropdown open for multiple selections
        }
    }

    function onToggle() {
        setIsOpen(!isOpen);
        if (!isOpen) {
            setInputValue('');
            setFocusedItemIndex(-1);
        }
    }

    function onClear() {
        onChange([]);
        setInputValue('');
        setFocusedItemIndex(-1);
    }

    function onInputChange(_event: React.FormEvent<HTMLInputElement>, text: string) {
        setInputValue(text);
        setFocusedItemIndex(-1); // Reset focus when typing
        if (!isOpen) {
            setIsOpen(true);
        }
    }

    function onKeyDown(event: React.KeyboardEvent) {
        if (event.key === 'Escape') {
            setIsOpen(false);
            setInputValue('');
            setFocusedItemIndex(-1);
        } else if (event.key === 'Enter' && isOpen) {
            event.preventDefault();
            if (focusedItemIndex >= 0 && focusedItemIndex < filteredOptions.length) {
                const selectedOption = filteredOptions[focusedItemIndex];
                onSelect(undefined, selectedOption.value);
            }
        } else if (event.key === 'ArrowDown' && isOpen) {
            event.preventDefault();
            setFocusedItemIndex((prev) => (prev < filteredOptions.length - 1 ? prev + 1 : 0));
        } else if (event.key === 'ArrowUp' && isOpen) {
            event.preventDefault();
            setFocusedItemIndex((prev) => (prev > 0 ? prev - 1 : filteredOptions.length - 1));
        }
    }

    // Filter options based on input
    const filteredOptions = useMemo(() => {
        return inputValue
            ? options.filter((option) => {
                  const label = option.label || option.value;
                  return label.toLowerCase().includes(inputValue.toLowerCase());
              })
            : options;
    }, [options, inputValue]);

    const hasResults = filteredOptions.length > 0;

    // Convert selections to Set for O(1) lookup performance
    const selectionsSet = useMemo(() => new Set(selections), [selections]);

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            variant="typeahead"
            onClick={onToggle}
            isExpanded={isOpen}
            isDisabled={isDisabled}
            aria-label={toggleAriaLabel}
            id={menuToggleId}
            className="pf-v5-u-w-100"
        >
            <TextInputGroup>
                <TextInputGroupMain
                    value={inputValue}
                    placeholder={
                        selections.length > 0 && !isOpen
                            ? `${selections.length} selected`
                            : placeholder
                    }
                    onChange={onInputChange}
                    onFocus={onToggle}
                    onKeyDown={onKeyDown}
                    autoComplete="off"
                    id={`${id}-select-typeahead`}
                />
                {selections.length > 0 && (
                    <TextInputGroupUtilities>
                        <Button
                            variant="plain"
                            onClick={(event) => {
                                event.stopPropagation();
                                onClear();
                            }}
                            aria-label="Clear selections"
                        >
                            <TimesIcon />
                        </Button>
                    </TextInputGroupUtilities>
                )}
            </TextInputGroup>
        </MenuToggle>
    );

    return (
        <Select
            id={id}
            isOpen={isOpen}
            selected={selections}
            onSelect={onSelect}
            onOpenChange={(nextOpen: boolean) => setIsOpen(nextOpen)}
            toggle={toggle}
            shouldFocusToggleOnSelect
            popperProps={
                menuAppendTo
                    ? {
                          appendTo: menuAppendTo,
                          direction,
                          maxWidth: 'trigger',
                      }
                    : {
                          direction,
                          maxWidth: 'trigger',
                      }
            }
            onBlur={onBlur}
        >
            <SelectList style={{ maxHeight, minWidth: '200px' }}>
                {hasResults ? (
                    filteredOptions.map((option, index) => (
                        <SelectOption
                            key={option.value}
                            value={option.value}
                            isDisabled={option.disabled}
                            isFocused={index === focusedItemIndex}
                            hasCheckbox
                            isSelected={selectionsSet.has(option.value)}
                        >
                            {option.label || option.value}
                        </SelectOption>
                    ))
                ) : (
                    <SelectOption isDisabled>No results found</SelectOption>
                )}
            </SelectList>
        </Select>
    );
}

export default TypeaheadCheckboxSelect;
