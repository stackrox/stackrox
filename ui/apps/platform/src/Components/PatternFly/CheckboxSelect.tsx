import React, { ReactElement, ReactNode, useState, useRef, useMemo } from 'react';
import {
    Select,
    SelectOption,
    SelectOptionProps,
    SelectGroup,
    MenuToggle,
    MenuToggleElement,
    Badge,
    Flex,
    FlexItem,
    SelectList,
    SelectPopperProps,
} from '@patternfly/react-core';

// Enhance children to automatically inject hasCheckbox and isSelected props
function enhanceSelectOptions(children: ReactNode, selectionsSet: Set<string>): ReactNode {
    return React.Children.map(children, (child) => {
        if (React.isValidElement(child)) {
            if (child.type === SelectOption) {
                const { value } = child.props;
                if (value != null) {
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
    onBlur?: React.FocusEventHandler<HTMLDivElement>;
    ariaLabel: string;
    children: ReactElement<SelectOptionProps>[];
    placeholderText?: string;
    toggleIcon?: ReactElement;
    toggleId?: string;
    menuAppendTo?: () => HTMLElement;
    isDisabled?: boolean;
    position?: SelectPopperProps['position'];
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
    menuAppendTo = undefined,
    isDisabled = false,
    position = undefined,
}: CheckboxSelectProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);
    const selectRef = useRef<HTMLDivElement>(null);

    function onToggle() {
        setIsOpen(!isOpen);
    }

    function handleBlur(event: React.FocusEvent<HTMLDivElement>) {
        const { currentTarget, relatedTarget } = event;

        // Wait for focus to settle, then check if it moved outside the component
        setTimeout(() => {
            let focusMovedOutside =
                !relatedTarget || !currentTarget.contains(relatedTarget as Node);

            // If menuAppendTo is used, also check if focus is within the appended menu container
            if (focusMovedOutside && menuAppendTo && relatedTarget) {
                const appendedContainer = menuAppendTo();
                focusMovedOutside = !appendedContainer.contains(relatedTarget as Node);
            }

            if (focusMovedOutside) {
                onBlur?.(event);
                setIsOpen(false);
            }
        }, 0);
    }

    function onSelect(
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
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

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
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
                popperProps={{
                    appendTo: menuAppendTo,
                    position,
                }}
            >
                <SelectList>{enhancedChildren}</SelectList>
            </Select>
        </div>
    );
}

export default CheckboxSelect;
