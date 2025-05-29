import React, { ReactElement } from 'react';
import {
    Badge,
    Divider,
    Dropdown,
    DropdownItem,
    DropdownList,
    MenuToggle,
    MenuToggleElement,
} from '@patternfly/react-core';

type Props = {
    selectedCount: number;
    isOpen: boolean;
    setOpen: (o: boolean) => void;
    onClear: () => void;
    children: ReactElement<typeof DropdownItem> | ReactElement<typeof DropdownItem>[];
};

export function FlowBulkDropdown({ selectedCount, isOpen, setOpen, onClear, children }: Props) {
    return (
        <Dropdown
            isOpen={isOpen}
            onOpenChange={setOpen}
            toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle
                    ref={ref}
                    isExpanded={isOpen}
                    badge={selectedCount > 0 ? <Badge isRead>{selectedCount}</Badge> : undefined}
                    isDisabled={selectedCount === 0}
                    onClick={() => setOpen(!isOpen)}
                >
                    Bulk actions
                </MenuToggle>
            )}
            onSelect={() => setOpen(false)}
        >
            <DropdownList>
                {children}
                <Divider component="li" />
                <DropdownItem onClick={onClear}>Clear selections</DropdownItem>
            </DropdownList>
        </Dropdown>
    );
}
