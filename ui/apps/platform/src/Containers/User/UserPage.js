import React from 'react';
import { NavLink, Route, Switch } from 'react-router-dom';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector, createSelector } from 'reselect';
import {
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    EmptyState,
    EmptyStateBody,
    Flex,
    FlexItem,
    Nav,
    NavExpandable,
    NavItem,
    NavList,
    PageSection,
    Title,
    EmptyStateHeader,
} from '@patternfly/react-core';

import DescriptionListCompact from 'Components/DescriptionListCompact';
import { selectors } from 'reducers';
import { userBasePath, userRolePath } from 'routePaths';
import User from 'utils/User';

import UserPermissionsForRolesTable from './UserPermissionsForRolesTable';
import UserPermissionsTable from './UserPermissionsTable';

const spacerPageSection = 'var(--pf-v5-global--spacer--md)';

const stylePageSection = {
    '--pf-v5-c-page__main-section--PaddingTop': spacerPageSection,
    '--pf-v5-c-page__main-section--PaddingRight': spacerPageSection,
    '--pf-v5-c-page__main-section--PaddingBottom': spacerPageSection,
    '--pf-v5-c-page__main-section--PaddingLeft': spacerPageSection,
};

const getUserRolePath = (roleName) => `${userBasePath}/roles/${roleName}`;

function UserPage({ resourceToAccessByRole, userData }) {
    const { email, name, roles, usedAuthProvider } = new User(userData);
    const authProviderName =
        usedAuthProvider?.type === 'basic' ? 'Basic' : usedAuthProvider?.name ?? '';

    return (
        <>
            <PageSection variant="light" style={stylePageSection}>
                <Title headingLevel="h1">User Profile</Title>
            </PageSection>
            <PageSection variant="light" style={stylePageSection}>
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
                        <DescriptionListTerm className="whitespace-nowrap">
                            Auth provider
                        </DescriptionListTerm>
                        <DescriptionListDescription>{authProviderName}</DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionListCompact>
            </PageSection>
            <PageSection variant="light" style={stylePageSection} isFilled>
                <Flex>
                    <FlexItem>
                        <div className="pf-v5-u-background-color-200">
                            <Nav aria-label="Roles" theme="light">
                                <NavList>
                                    <NavItem>
                                        <NavLink
                                            exact
                                            to={userBasePath}
                                            activeClassName="pf-m-current"
                                        >
                                            User permissions for roles
                                        </NavLink>
                                    </NavItem>
                                    <NavExpandable title="User roles" isExpanded>
                                        {roles.map((role) => (
                                            <NavItem key={role.name} activeClassName="pf-m-current">
                                                <NavLink
                                                    exact
                                                    to={getUserRolePath(role.name)}
                                                    activeClassName="pf-m-current"
                                                >
                                                    {role.name}
                                                </NavLink>
                                            </NavItem>
                                        ))}
                                    </NavExpandable>
                                </NavList>
                            </Nav>
                        </div>
                    </FlexItem>
                    <FlexItem>
                        <Switch>
                            <Route
                                path={userRolePath}
                                render={({
                                    match: {
                                        params: { roleName },
                                    },
                                }) => {
                                    const role = roles.find((_role) => _role.name === roleName);

                                    if (role) {
                                        return (
                                            <UserPermissionsTable
                                                permissions={role?.resourceToAccess ?? {}}
                                            />
                                        );
                                    }

                                    return (
                                        <EmptyState>
                                            <EmptyStateHeader
                                                titleText="Role not found for user"
                                                headingLevel="h4"
                                            />
                                            <EmptyStateBody>{`Role name: ${roleName}`}</EmptyStateBody>
                                        </EmptyState>
                                    );
                                }}
                            />
                            <Route path={userBasePath}>
                                <UserPermissionsForRolesTable
                                    resourceToAccessByRole={resourceToAccessByRole}
                                />
                            </Route>
                        </Switch>
                    </FlexItem>
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
