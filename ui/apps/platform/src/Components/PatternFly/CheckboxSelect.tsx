import React, { ReactElement, ReactNode, useState } from 'react';
import {
    Select,
    SelectOption,
    SelectOptionProps,
    SelectGroup,
    SelectGroupProps,
    MenuToggle,
    MenuToggleElement,
    Badge,
    Flex,
    FlexItem,
    SelectList,
} from '@patternfly/react-core';

// Type for SelectGroup that contains SelectOption children
type SelectGroupWithOptions = ReactElement<
    SelectGroupProps & {
        children: ReactElement<SelectOptionProps>[];
    }
>;

export type CheckboxSelectProps = {
    id?: string;
    selections: string[];
    onChange: (selection: string[]) => void;
    onBlur?: React.FocusEventHandler<HTMLDivElement>;
    ariaLabel: string;
    children: (ReactElement<SelectOptionProps> | SelectGroupWithOptions)[];
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

    // Recursively enhance children to automatically inject hasCheckbox and isSelected props
    // Handles both flat SelectOption children and SelectOption nested within SelectGroup
    const enhanceSelectOptions = (children: ReactNode): ReactNode => {
        return React.Children.map(children, (child) => {
            if (React.isValidElement(child)) {
                if (child.type === SelectOption) {
                    // Direct SelectOption - inject props
                    const { value } = child.props;
                    if (value != null) {
                        return React.cloneElement(child, {
                            hasCheckbox: true,
                            isSelected: selections.includes(value as string),
                            ...child.props, // Allow explicit overrides if needed
                        });
                    }
                } else if (child.type === SelectGroup) {
                    // SelectGroup - recursively process its children
                    return React.cloneElement(child, {
                        ...child.props,
                        children: enhanceSelectOptions(child.props.children),
                    });
                }
            }
            return child;
        });
    };

    const enhancedChildren = enhanceSelectOptions(children);

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
