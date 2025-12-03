import { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import {
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    EmptyState,
    EmptyStateBody,
    PageSection,
    Tab,
    TabContent,
    TabContentBody,
    TabTitleText,
    Tabs,
    Title,
} from '@patternfly/react-core';

import DescriptionListCompact from 'Components/DescriptionListCompact';
import { selectors } from 'reducers';
import User from 'utils/User';

import UserPermissionsForRolesTable from './UserPermissionsForRolesTable';
import UserPermissionsTable from './UserPermissionsTable';

function UserPage({ resourceToAccessByRole, userData }) {
    const { email, name, roles, usedAuthProvider } = new User(userData);
    const authProviderName =
        usedAuthProvider?.type === 'basic' ? 'Basic' : (usedAuthProvider?.name ?? '');

    const [activeTabKey, setActiveTabKey] = useState(0);
    const [activeRoleTabKey, setActiveRoleTabKey] = useState(0);

    const handleTabClick = (_, tabIndex) => {
        setActiveTabKey(tabIndex);
    };

    const handleRoleTabClick = (_, tabIndex) => {
        setActiveRoleTabKey(tabIndex);
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
                <Tabs isVertical activeTabKey={activeTabKey} onSelect={handleTabClick}>
                    <Tab
                        eventKey={0}
                        title={<TabTitleText>User permissions for roles</TabTitleText>}
                    >
                        <TabContent>
                            <TabContentBody>
                                <UserPermissionsForRolesTable
                                    resourceToAccessByRole={resourceToAccessByRole}
                                />
                            </TabContentBody>
                        </TabContent>
                    </Tab>

                    <Tab eventKey={1} title={<TabTitleText>User roles</TabTitleText>}>
                        <TabContent>
                            <TabContentBody>
                                {roles.length > 0 ? (
                                    <Tabs
                                        isSecondary
                                        activeTabKey={activeRoleTabKey}
                                        onSelect={handleRoleTabClick}
                                    >
                                        {roles.map((role, index) => (
                                            <Tab
                                                key={role.name}
                                                eventKey={index}
                                                title={<TabTitleText>{role.name}</TabTitleText>}
                                            >
                                                <TabContent>
                                                    <TabContentBody>
                                                        <UserPermissionsTable
                                                            permissions={
                                                                role?.resourceToAccess ?? {}
                                                            }
                                                        />
                                                    </TabContentBody>
                                                </TabContent>
                                            </Tab>
                                        ))}
                                    </Tabs>
                                ) : (
                                    <EmptyState
                                        headingLevel="h4"
                                        titleText="No roles assigned to user"
                                    >
                                        <EmptyStateBody>User has no roles assigned</EmptyStateBody>
                                    </EmptyState>
                                )}
                            </TabContentBody>
                        </TabContent>
                    </Tab>
                </Tabs>
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
