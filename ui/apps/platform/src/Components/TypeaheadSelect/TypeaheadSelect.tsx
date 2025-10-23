import { useState } from 'react';
import type {
    FocusEventHandler,
    FormEvent,
    KeyboardEvent,
    MouseEvent as ReactMouseEvent,
    ReactElement,
    ReactNode,
    Ref,
} from 'react';
import {
    Select,
    SelectOption,
    MenuToggle,
    SelectList,
    TextInputGroup,
    TextInputGroupMain,
    MenuFooter,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';

export type TypeaheadSelectOption = {
    value: string;
    label?: string;
    disabled?: boolean;
};

export type TypeaheadSelectProps = {
    id: string;
    menuToggleId?: string;
    value: string;
    onChange: (value: string) => void;
    options: TypeaheadSelectOption[];
    allowCreate?: boolean;
    placeholder?: string;
    isDisabled?: boolean;
    toggleAriaLabel?: string;
    onBlur?: FocusEventHandler<HTMLDivElement>;
    menuAppendTo?: () => HTMLElement;
    footer?: ReactNode;
    maxHeight?: string;
    direction?: 'up' | 'down';
    className?: string;
};

function TypeaheadSelect({
    id,
    menuToggleId,
    value,
    onChange,
    options,
    allowCreate = false,
    placeholder = 'Type to search...',
    isDisabled = false,
    toggleAriaLabel,
    onBlur,
    menuAppendTo,
    footer,
    maxHeight = '300px',
    direction = 'down',
    className,
}: TypeaheadSelectProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);
    const [inputValue, setInputValue] = useState('');
    const [focusedItemIndex, setFocusedItemIndex] = useState<number>(-1);

    function onSelect(
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        selection: string | number | undefined
    ) {
        if (typeof selection === 'string') {
            setIsOpen(false);
            onChange(selection);
            setInputValue('');
        }
    }

    function onToggle() {
        setIsOpen(!isOpen);
    }

    function onInputChange(_event: FormEvent<HTMLInputElement>, text: string) {
        setInputValue(text);
        setFocusedItemIndex(-1); // Reset focus when typing
        if (!isOpen) {
            setIsOpen(true);
        }
    }

    function onKeyDown(event: KeyboardEvent) {
        const allOptions = shouldShowCreateOption
            ? [...filteredOptions, { value: inputValue, label: `Create "${inputValue}"` }]
            : filteredOptions;

        if (event.key === 'Escape') {
            setIsOpen(false);
            setInputValue('');
            setFocusedItemIndex(-1);
        } else if (event.key === 'Enter' && isOpen) {
            event.preventDefault();
            if (focusedItemIndex >= 0 && focusedItemIndex < allOptions.length) {
                const selectedOption = allOptions[focusedItemIndex];
                onChange(selectedOption.value);
                setIsOpen(false);
                setInputValue('');
                setFocusedItemIndex(-1);
            }
        } else if (event.key === 'ArrowDown' && isOpen) {
            event.preventDefault();
            setFocusedItemIndex((prev) => (prev < allOptions.length - 1 ? prev + 1 : 0));
        } else if (event.key === 'ArrowUp' && isOpen) {
            event.preventDefault();
            setFocusedItemIndex((prev) => (prev > 0 ? prev - 1 : allOptions.length - 1));
        }
    }

    // Filter options based on input
    const filteredOptions = inputValue
        ? options.filter((option) => {
              const label = option.label || option.value;
              return label.toLowerCase().includes(inputValue.toLowerCase());
          })
        : options;

    // Check if we should show create option
    const shouldShowCreateOption =
        allowCreate &&
        inputValue &&
        !options.some((option) => {
            const label = option.label || option.value;
            return label.toLowerCase() === inputValue.toLowerCase();
        });

    const hasResults = filteredOptions.length > 0 || shouldShowCreateOption;

    // Get display text for the selected value
    const getDisplayValue = (): string => {
        if (!value) {
            return '';
        }
        const selectedOption = options.find((option) => option.value === value);
        return selectedOption?.label || selectedOption?.value || value;
    };

    const toggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            variant="typeahead"
            onClick={onToggle}
            isExpanded={isOpen}
            isDisabled={isDisabled}
            aria-label={toggleAriaLabel}
            id={menuToggleId}
            className={className}
        >
            <TextInputGroup>
                <TextInputGroupMain
                    value={isOpen ? inputValue : getDisplayValue()}
                    placeholder={placeholder}
                    onChange={onInputChange}
                    onFocus={onToggle}
                    onKeyDown={onKeyDown}
                    autoComplete="off"
                    id={`${id}-select-typeahead`}
                />
            </TextInputGroup>
        </MenuToggle>
    );

    return (
        <Select
            id={id}
            isOpen={isOpen}
            selected={value}
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
            <SelectList style={{ maxHeight }}>
                {hasResults ? (
                    <>
                        {filteredOptions.map((option, index) => (
                            <SelectOption
                                key={option.value}
                                value={option.value}
                                isDisabled={option.disabled}
                                isFocused={index === focusedItemIndex}
                            >
                                {option.label || option.value}
                            </SelectOption>
                        ))}
                        {shouldShowCreateOption && (
                            <SelectOption
                                value={inputValue}
                                isFocused={filteredOptions.length === focusedItemIndex}
                            >
                                Create &quot;{inputValue}&quot;
                            </SelectOption>
                        )}
                    </>
                ) : (
                    <SelectOption isDisabled>No results found</SelectOption>
                )}
            </SelectList>
            {footer && <MenuFooter>{footer}</MenuFooter>}
        </Select>
    );
}

export default TypeaheadSelect;
