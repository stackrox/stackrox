import { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import {
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    EmptyState,
    EmptyStateBody,
    Flex,
    FlexItem,
    PageSection,
    SelectGroup,
    SelectOption,
    Title,
} from '@patternfly/react-core';

import DescriptionListCompact from 'Components/DescriptionListCompact';
import SelectSingle from 'Components/SelectSingle/SelectSingle';
import { selectors } from 'reducers';
import User from 'utils/User';

import UserPermissionsForRolesTable from './UserPermissionsForRolesTable';
import UserPermissionsTable from './UserPermissionsTable';

function UserPage({ resourceToAccessByRole, userData }) {
    const { email, name, roles, usedAuthProvider } = new User(userData);
    const authProviderName =
        usedAuthProvider?.type === 'basic' ? 'Basic' : (usedAuthProvider?.name ?? '');

    const [selectedRole, setSelectedRole] = useState('ALL');

    const handleRoleSelect = (_, selection) => {
        setSelectedRole(selection);
    };

    return (
        <>
            <PageSection hasBodyWrapper={false}>
                <Title headingLevel="h1">User Profile</Title>
            </PageSection>
            <PageSection hasBodyWrapper={false}>
                <DescriptionListCompact isHorizontal>
                    <DescriptionListGroup>
                        <DescriptionListTerm>User name</DescriptionListTerm>
                        <DescriptionListDescription>{name}</DescriptionListDescription>
                    </DescriptionListGroup>
                    {email && (
                        <DescriptionListGroup>
                            <DescriptionListTerm>User email</DescriptionListTerm>
                            <DescriptionListDescription>{email}</DescriptionListDescription>
                        </DescriptionListGroup>
                    )}
                    <DescriptionListGroup>
                        <DescriptionListTerm className="pf-v6-u-text-nowrap">
                            Auth provider
                        </DescriptionListTerm>
                        <DescriptionListDescription>{authProviderName}</DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionListCompact>
            </PageSection>
            <PageSection hasBodyWrapper={false} isFilled>
                <Flex direction={{ default: 'column' }} gap={{ default: 'gapMd' }}>
                    <FlexItem>
                        <Flex direction={{ default: 'column' }} gap={{ default: 'gapMd' }}>
                            <FlexItem>
                                <SelectSingle
                                    id="user-role-selector"
                                    value={selectedRole}
                                    handleSelect={handleRoleSelect}
                                    placeholderText="Select a view"
                                    isFullWidth={false}
                                >
                                    {[
                                        <SelectOption
                                            key="ALL"
                                            value="ALL"
                                            description="View aggregated permissions across all assigned roles"
                                        >
                                            User permissions for roles
                                        </SelectOption>,
                                        <Divider key="divider" component="li" />,
                                        <SelectGroup key="roles-group" label="User roles">
                                            {roles.map((role) => (
                                                <SelectOption key={role.name} value={role.name}>
                                                    {role.name}
                                                </SelectOption>
                                            ))}
                                        </SelectGroup>,
                                    ]}
                                </SelectSingle>
                            </FlexItem>

                            {selectedRole === 'ALL' && (
                                <FlexItem>
                                    <UserPermissionsForRolesTable
                                        resourceToAccessByRole={resourceToAccessByRole}
                                    />
                                </FlexItem>
                            )}

                            {selectedRole && selectedRole !== 'ALL' && (
                                <FlexItem>
                                    <UserPermissionsTable
                                        permissions={
                                            roles.find((r) => r.name === selectedRole)
                                                ?.resourceToAccess ?? {}
                                        }
                                    />
                                </FlexItem>
                            )}
                        </Flex>
                    </FlexItem>

                    {roles.length === 0 && (
                        <FlexItem>
                            <EmptyState headingLevel="h4" titleText="No roles assigned to user">
                                <EmptyStateBody>User has no roles assigned</EmptyStateBody>
                            </EmptyState>
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
        </>
    );
}

UserPage.propTypes = {
    userData: PropTypes.shape({
        userAttributes: PropTypes.arrayOf(PropTypes.shape({})),
        userInfo: PropTypes.shape({
            roles: PropTypes.arrayOf(PropTypes.shape({})),
            permissions: PropTypes.shape({}),
        }),
    }).isRequired,
    resourceToAccessByRole: PropTypes.objectOf(
        PropTypes.shape({
            read: PropTypes.arrayOf(PropTypes.string).isRequired,
            write: PropTypes.arrayOf(PropTypes.string).isRequired,
        })
    ).isRequired,
};

const resourceToAccessByRoleSelector = createSelector(
    [selectors.getCurrentUser],
    (userData) => new User(userData).resourceToAccessByRole
);

const mapStateToProps = createStructuredSelector({
    userData: selectors.getCurrentUser,
    resourceToAccessByRole: resourceToAccessByRoleSelector,
});

export default connect(mapStateToProps, null)(UserPage);
