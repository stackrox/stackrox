import { useMemo, useState } from 'react';
import type { Ref } from 'react';
import {
    Badge,
    Button,
    Divider,
    Flex,
    FlexItem,
    Menu,
    MenuContent,
    MenuFooter,
    MenuItem,
    MenuList,
    MenuSearch,
    MenuSearchInput,
    MenuToggle,
    SearchInput,
    Select,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

type RoleSelectorProps = {
    id?: string;
    roles?: string[];
    selectedRoles?: string[];
    isEditable: boolean;
    isGenerated: boolean;
    isRolesLoading: boolean;
    onRoleSelect: (id, value) => void;
    onRoleSelectionClear: () => void;
};

function RoleSelector({
    id,
    roles = [],
    selectedRoles = [],
    isEditable,
    isGenerated,
    isRolesLoading,
    onRoleSelect,
    onRoleSelectionClear,
}: RoleSelectorProps) {
    const { isOpen: isRoleOpen, toggleSelect: toggleIsRoleOpen } = useSelectToggle();
    const [input, setInput] = useState('');

    const handleTextInputChange = (value: string) => {
        setInput(value);
    };

    const filteredRoleSelectMenuItems = useMemo(() => {
        return roles
            .filter((roleName) => roleName.toLowerCase().includes(input.toString().toLowerCase()))
            .map((roleName) => {
                return (
                    <MenuItem
                        key={roleName}
                        hasCheckbox
                        itemId={roleName}
                        isSelected={selectedRoles.includes(roleName)}
                        isDisabled={!isEditable || isRolesLoading || isGenerated}
                    >
                        <span className="pf-v5-u-mx-xs" data-testid="namespace-name">
                            {roleName}
                        </span>
                    </MenuItem>
                );
            });
    }, [roles, input, isEditable, isGenerated, isRolesLoading, selectedRoles]);

    const roleSelectMenu = (
        <Menu onSelect={onRoleSelect} selected={selectedRoles}>
            <MenuSearch>
                <MenuSearchInput>
                    <SearchInput
                        value={input}
                        aria-label="Filter roles"
                        placeholder="Filter roles..."
                        onChange={(_event, value) => handleTextInputChange(value)}
                    />
                </MenuSearchInput>
            </MenuSearch>
            <Divider className="pf-v5-u-m-0" />
            <MenuContent>
                <MenuList>
                    {filteredRoleSelectMenuItems.length === 0 && (
                        <MenuItem isDisabled key="no result">
                            No roles found
                        </MenuItem>
                    )}
                    {filteredRoleSelectMenuItems}
                </MenuList>
            </MenuContent>
            <MenuFooter>
                <Button
                    variant="link"
                    isInline
                    onClick={onRoleSelectionClear}
                    isDisabled={selectedRoles.length === 0 || !isEditable || isGenerated}
                >
                    Clear selections
                </Button>
            </MenuFooter>
        </Menu>
    );

    const toggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            id={id}
            onClick={() => toggleIsRoleOpen(!isRoleOpen)}
            isExpanded={isRoleOpen}
            isDisabled={roles.length === 0 || !isEditable || isGenerated}
            aria-label={'Select roles'}
            className={'role-select'}
            variant={'plainText'}
        >
            <Flex alignSelf={{ default: 'alignSelfCenter' }}>
                <FlexItem spacer={{ default: 'spacerSm' }}>
                    <span style={{ position: 'relative', top: '1px' }}>
                        {roles.length === 0 ? 'No roles' : 'Roles'}
                    </span>
                </FlexItem>
                {selectedRoles.length !== 0 && (
                    <FlexItem spacer={{ default: 'spacerSm' }}>
                        <Badge isRead>{selectedRoles.length}</Badge>
                    </FlexItem>
                )}
            </Flex>
        </MenuToggle>
    );

    return (
        <Select
            isOpen={isRoleOpen}
            onOpenChange={(nextOpen: boolean) => toggleIsRoleOpen(nextOpen)}
            toggle={toggle}
            popperProps={{
                maxWidth: '400px',
                direction: 'down',
            }}
        >
            {roleSelectMenu}
        </Select>
    );
}

export default RoleSelector;
