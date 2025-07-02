import React, { CSSProperties } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Divider, DropdownItem } from '@patternfly/react-core';

import useAnalytics, { INVITE_USERS_MODAL_OPENED } from 'hooks/useAnalytics';
import usePermissions from 'hooks/usePermissions';
import { selectors } from 'reducers';
import { actions as authActions } from 'reducers/auth';
import { actions as inviteActions } from 'reducers/invite';
import { userBasePath } from 'routePaths';
import User from 'utils/User';
import MenuDropdown from 'Components/PatternFly/MenuDropdown';

const userMenuStyleConstant = {
    '--pf-v5-u-min-width--MinWidth': '20ch',
    pointerEvents: 'none',
} as CSSProperties;

function UserMenu({ logout, setInviteModalVisibility, userData }) {
    const { analyticsTrack } = useAnalytics();
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForInviting = hasReadWriteAccess('Access');

    function onClickInviteUsers() {
        // track request to invite
        analyticsTrack(INVITE_USERS_MODAL_OPENED);

        setInviteModalVisibility(true);
    }

    const user = new User(userData);
    const { email, name, roles } = user;

    const displayName = email ? (
        <span>
            <span data-testid="menu-user-name">{name}</span> (
            <span data-testid="menu-user-email">{email}</span>)
        </span>
    ) : (
        <span data-testid="menu-user-name">{name}</span>
    );
    const displayRoles = Array.isArray(roles) ? roles.map((role) => role.name).join(',') : '';

    return (
        <MenuDropdown
            popperProps={{ position: 'end' }}
            toggleText={name ? name : '--'}
            toggleVariant="plainText"
        >
            <DropdownItem
                key="user"
                description={<span data-testid="menu-user-roles">{displayRoles}</span>}
                className="pf-v5-u-min-width"
                style={userMenuStyleConstant}
            >
                {displayName}
            </DropdownItem>
            <Divider component="li" key="separator" />
            <DropdownItem key="profile" href={userBasePath}>
                My profile
            </DropdownItem>
            {hasWriteAccessForInviting && (
                <DropdownItem key="open-invite" onClick={onClickInviteUsers}>
                    Invite users
                </DropdownItem>
            )}
            <DropdownItem key="logout" component="button" onClick={logout}>
                Log out
            </DropdownItem>
        </MenuDropdown>
    );
}

UserMenu.propTypes = {
    logout: PropTypes.func.isRequired,
    setInviteModalVisibility: PropTypes.func.isRequired,
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
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    setInviteModalVisibility: (show) => dispatch(inviteActions.setInviteModalVisibility(show)),
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    logout: () => dispatch(authActions.logout()),
});

export default connect(mapStateToProps, mapDispatchToProps)(UserMenu);
