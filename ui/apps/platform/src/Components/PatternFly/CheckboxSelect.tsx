import React, { ReactElement, useState, useRef } from 'react';
import {
    Select,
    SelectOption,
    SelectOptionProps,
    MenuToggle,
    MenuToggleElement,
    Badge,
    Flex,
    FlexItem,
    SelectList,
} from '@patternfly/react-core';

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
    menuAppendTo,
    isDisabled = false,
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

    // Automatically inject hasCheckbox and isSelected props
    const enhancedChildren = React.Children.map(children, (child) => {
        if (React.isValidElement(child) && child.type === SelectOption) {
            const { value } = child.props;
            if (value != null) {
                return React.cloneElement(child, {
                    hasCheckbox: true,
                    isSelected: selections.includes(value as string),
                    ...child.props, // Allow explicit overrides if needed
                });
            }
        }
        return child;
    });

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
                popperProps={
                    menuAppendTo
                        ? {
                              appendTo: menuAppendTo,
                          }
                        : undefined
                }
            >
                <SelectList>{enhancedChildren}</SelectList>
            </Select>
        </div>
    );
}

export default CheckboxSelect;
