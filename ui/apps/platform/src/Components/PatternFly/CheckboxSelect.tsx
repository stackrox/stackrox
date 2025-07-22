import React, { ReactElement, useState } from 'react';
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
}: CheckboxSelectProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);

    function onToggle() {
        setIsOpen(!isOpen);
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
        <Select
            id={id}
            aria-label={ariaLabel}
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
                      }
                    : undefined
            }
            onBlur={onBlur}
        >
            <SelectList>{enhancedChildren}</SelectList>
        </Select>
    );
}

export default CheckboxSelect;
