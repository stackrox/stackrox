import React, { useMemo } from 'react';
import {
    Badge,
    Button,
    Divider,
    Flex,
    FlexItem,
    Menu,
    MenuContent,
    MenuFooter,
    MenuInput,
    MenuItem,
    MenuList,
    SearchInput,
    Select,
} from '@patternfly/react-core';
import useSelectToggle from "hooks/patternfly/useSelectToggle";
import { Role } from 'services/RolesService';

type RoleSelectorProps = {
    roles?: Role[];
    selectedRoles?: string[];
    onRoleSelect: (id, value) => void;
    onRoleSelectionClear: () => void;
};

function RoleSelector({
    roles = [],
    selectedRoles = [],
    onRoleSelect,
    onRoleSelectionClear,
}: RoleSelectorProps) {
    const {isOpen: isRoleOpen, toggleSelect: toggleIsRoleOpen} = useSelectToggle();
    const [input, setInput] = React.useState('');

    const handleTextInputChange = (value: string) => {
        setInput(value);
    };

    const filteredRoleSelectMenuItems = useMemo(() => {
        const roleSelectMenuItems = roles
            .filter((role) =>
                role.name.toLowerCase().includes(input.toString().toLowerCase())
            )
            .map((role) => {
                return (
                    <MenuItem
                        key={role.name}
                        hasCheck={true}
                        itemId={role.name}
                        isSelected={selectedRoles.includes(role.name)}
                    >
                        <span className="pf-u-mx-xs" data-testid="namespace-name">
                                {role.name}
                        </span>
                    </MenuItem>
                );
            });

        return roleSelectMenuItems;
    }, [roles, input, selectedRoles]);


    const roleSelectMenu = (
        <Menu onSelect={onRoleSelect} selected={selectedRoles} isScrollable>
            <MenuInput className="pf-u-p-md">
                <SearchInput
                    value={input}
                    aria-label="Filter roles"
                    type="search"
                    placeholder="Filter roles..."
                    onChange={(_event, value) => handleTextInputChange(value)}
                />
            </MenuInput>
            <Divider className="pf-u-m-0" />
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
                    isDisabled={selectedRoles.length === 0}
                >
                    Clear selections
                </Button>
            </MenuFooter>
        </Menu>
    );

    return (
        <Select
            isOpen={isRoleOpen}
            onToggle={toggleIsRoleOpen}
            className="role-select"
            placeholderText={
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
            }
            toggleAriaLabel="Select roles"
            isDisabled={roles.length === 0}
            isPlain
            customContent={roleSelectMenu}
        />
    );
}

export default RoleSelector;

