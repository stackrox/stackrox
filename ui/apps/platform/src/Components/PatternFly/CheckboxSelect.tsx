import React, { useState, useRef, useMemo } from 'react';
import type {
    FocusEvent,
    FocusEventHandler,
    MouseEvent as ReactMouseEvent,
    ReactElement,
    ReactNode,
    Ref,
} from 'react';
import {
    Select,
    SelectOption,
    SelectGroup,
    MenuToggle,
    Badge,
    Flex,
    FlexItem,
    SelectList,
} from '@patternfly/react-core';
import type {
    MenuToggleElement,
    SelectOptionProps,
    SelectPopperProps,
} from '@patternfly/react-core';

// Enhance children to automatically inject hasCheckbox and isSelected props
function enhanceSelectOptions(children: ReactNode, selectionsSet: Set<string>): ReactNode {
    return React.Children.map(children, (child) => {
        if (React.isValidElement(child)) {
            if (child.type === SelectOption) {
                const { value } = child.props;
                if (value !== null && value !== undefined) {
                    return React.cloneElement(child, {
                        hasCheckbox: true,
                        isSelected: selectionsSet.has(value as string),
                        ...child.props, // Allow explicit overrides if needed
                    });
                }
            } else if (child.type === SelectGroup) {
                // Recursively enhance SelectOption children within SelectGroup
                const enhancedGroupChildren = enhanceSelectOptions(
                    child.props.children,
                    selectionsSet
                );
                return React.cloneElement(child, {
                    ...child.props,
                    children: enhancedGroupChildren,
                });
            }
        }
        return child;
    });
}

export type CheckboxSelectProps = {
    id?: string;
    selections: string[];
    onChange: (selection: string[]) => void;
    onBlur?: FocusEventHandler<HTMLDivElement>;
    ariaLabel: string;
    children: ReactElement<SelectOptionProps>[];
    placeholderText?: string;
    toggleIcon?: ReactElement;
    toggleId?: string;
    isDisabled?: boolean;
    popperProps?: SelectPopperProps;
};

function CheckboxSelect({
    id,
    selections,
    onChange,
    onBlur,
    ariaLabel,
    children,
    placeholderText = 'Filter by value',
    toggleIcon,
    toggleId,
    isDisabled = false,
    popperProps,
}: CheckboxSelectProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);
    const selectRef = useRef<HTMLDivElement>(null);

    function onToggle() {
        setIsOpen(!isOpen);
    }

    function handleBlur(event: FocusEvent<HTMLDivElement>) {
        const { currentTarget, relatedTarget } = event;

        // Wait for focus to settle, then check if it moved outside the component
        setTimeout(() => {
            let focusMovedOutside =
                !relatedTarget || !currentTarget.contains(relatedTarget as Node);

            // If popperProps.appendTo is used, also check if focus is within the appended menu container
            if (focusMovedOutside && popperProps?.appendTo && relatedTarget) {
                const { appendTo } = popperProps;
                if (typeof appendTo === 'function') {
                    const appendedContainer = appendTo();
                    focusMovedOutside = !appendedContainer.contains(relatedTarget as Node);
                } else if (appendTo instanceof HTMLElement) {
                    focusMovedOutside = !appendTo.contains(relatedTarget as Node);
                }
                // If appendTo is "inline", we don't need to check anything additional
            }

            if (focusMovedOutside) {
                onBlur?.(event);
                setIsOpen(false);
            }
        }, 0);
    }

    function onSelect(
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        selection: string | number | undefined
    ) {
        if (typeof selection !== 'string' || !selections || !onChange) {
            return;
        }
        if (selections.includes(selection)) {
            onChange(selections.filter((item) => item !== selection));
        } else {
            onChange([...selections, selection]);
        }
    }

    const toggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            className="pf-v5-u-w-100"
            id={toggleId}
            ref={toggleRef}
            onClick={onToggle}
            isExpanded={isOpen}
            isDisabled={isDisabled}
            icon={toggleIcon}
            aria-label={ariaLabel}
        >
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <FlexItem>{placeholderText}</FlexItem>
                {selections.length > 0 && <Badge isRead>{selections.length}</Badge>}
            </Flex>
        </MenuToggle>
    );

    // Convert selections to Set for O(1) lookup performance
    const selectionsSet = useMemo(() => new Set(selections), [selections]);

    // Enhance children to automatically inject hasCheckbox and isSelected props
    const enhancedChildren = useMemo(() => {
        return enhanceSelectOptions(children, selectionsSet);
    }, [children, selectionsSet]);

    return (
        <div ref={selectRef} onBlur={handleBlur}>
            <Select
                id={id}
                aria-label={ariaLabel}
                isOpen={isOpen}
                selected={selections}
                onSelect={onSelect}
                onOpenChange={(nextOpen: boolean) => {
                    setIsOpen(nextOpen);
                }}
                toggle={toggle}
                shouldFocusToggleOnSelect
                popperProps={popperProps}
            >
                <SelectList>{enhancedChildren}</SelectList>
            </Select>
        </div>
    );
}

export default CheckboxSelect;
