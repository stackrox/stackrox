import React, { ReactElement, useState } from 'react';
import {
    Select,
    SelectOption,
    MenuToggle,
    MenuToggleElement,
    Badge,
    Flex,
    FlexItem,
} from '@patternfly/react-core';

export type CheckboxSelectProps = {
    id?: string;
    selections: string[];
    onChange: (selection: string[]) => void;
    onBlur?: React.FocusEventHandler<HTMLDivElement>;
    ariaLabel: string;
    children: ReactElement<typeof SelectOption>[];
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

    return (
        <Select
            id={id}
            role="menu"
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
            {children}
        </Select>
    );
}

export default CheckboxSelect;
