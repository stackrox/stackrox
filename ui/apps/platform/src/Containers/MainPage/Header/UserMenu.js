import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import initials from 'initials';
import {
    Dropdown,
    DropdownItem,
    DropdownSeparator,
    DropdownToggle,
    Flex,
} from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as authActions } from 'reducers/auth';
import { userBasePath } from 'routePaths';
import User from 'utils/User';

function RoleChips({ roleNames }) {
    if (roleNames.length === 0) {
        return <span>No roles</span>;
    }

    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            {roleNames.map((roleName) => (
                <div className="pf-c-chip" key={roleName}>
                    <span className="pf-c-chip__text">{roleName}</span>
                </div>
            ))}
        </Flex>
    );
}

function UserItem({ email, name, roleNames }) {
    const thClassName = 'pf-u-font-weight-normal pf-u-pr-md pf-u-text-align-left pf-u-text-nowrap';

    return (
        <div>
            <div>User Profile</div>
            <table>
                <tbody>
                    <tr key="name">
                        <th scope="row" className={thClassName}>
                            User name
                        </th>
                        <td data-testid="menu-user-name">{name}</td>
                    </tr>
                    {email && (
                        <tr key="email">
                            <th scope="row" className={thClassName}>
                                User email
                            </th>
                            <td data-testid="menu-user-email">{email}</td>
                        </tr>
                    )}
                    <tr key="roles">
                        <th scope="row" className={thClassName}>
                            User roles
                        </th>
                        <td data-testid="menu-user-roles">
                            <RoleChips roleNames={roleNames} />
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>
    );
}

function UserMenu({ logout, userData }) {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect() {
        setIsOpen(false);
    }

    const user = new User(userData);
    const { email, name, roles } = user;

    const dropdownItems = [
        <DropdownItem key="user" href={userBasePath}>
            <UserItem email={email} name={name} roleNames={roles.map((role) => role.name)} />
        </DropdownItem>,
        <DropdownSeparator key="separator" />,
        <DropdownItem key="logout" component="button" onClick={logout}>
            Log out
        </DropdownItem>,
    ];

    const toggle = (
        <DropdownToggle aria-label="User menu" onToggle={setIsOpen} toggleIndicator={null}>
            <span className="h-10 w-10 flex items-center justify-center leading-none text-xl border border-base-400 rounded-full">
                {name ? initials(name) : '--'}
            </span>
        </DropdownToggle>
    );

    return (
        <Dropdown
            dropdownItems={dropdownItems}
            isOpen={isOpen}
            isPlain
            onSelect={onSelect}
            position="right"
            toggle={toggle}
        />
    );
}

UserMenu.propTypes = {
    logout: PropTypes.func.isRequired,
    userData: PropTypes.shape({
        userInfo: PropTypes.shape({
            username: PropTypes.string,
            friendlyName: PropTypes.string,
            roles: PropTypes.arrayOf(
                PropTypes.shape({
                    name: PropTypes.string,
                })
            ),
        }),
        userAttributes: PropTypes.arrayOf(PropTypes.shape({})),
    }).isRequired,
};

const mapStateToProps = createStructuredSelector({
    userData: selectors.getCurrentUser,
});

const mapDispatchToProps = (dispatch) => ({
    logout: () => dispatch(authActions.logout()),
});

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(UserMenu));
