import React, { ReactElement, useState } from 'react';
import {
    Select,
    SelectList,
    SelectOption,
    MenuToggle,
    MenuToggleElement,
    Badge,
    Flex,
    FlexItem,
} from '@patternfly/react-core';

export type CheckboxSelectProps = {
    id?: string;
    name?: string;
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
    name: _name, // eslint-disable-line @typescript-eslint/no-unused-vars -- Keep for backward compatibility
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

    function onToggleClick() {
        setIsOpen(!isOpen);
    }

    function onSelect(
        event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) {
        if (typeof value !== 'string' || !selections || !onChange) {
            return;
        }
        if (selections.includes(value)) {
            onChange(selections.filter((item) => item !== value));
        } else {
            onChange([...selections, value]);
        }
    }

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={onToggleClick}
            isExpanded={isOpen}
            id={toggleId}
            icon={toggleIcon}
        >
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <FlexItem>{placeholderText}</FlexItem>
                {selections && selections.length > 0 && <Badge isRead>{selections.length}</Badge>}
            </Flex>
        </MenuToggle>
    );

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
            onBlur={onBlur}
            popperProps={{
                appendTo: menuAppendTo,
            }}
        >
            <SelectList>{children}</SelectList>
        </Select>
    );
}

export default CheckboxSelect;
