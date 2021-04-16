import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { MoreHorizontal } from 'react-feather';
import { Avatar } from '@stackrox/ui-components';

import { selectors } from 'reducers';
import { actions as authActions } from 'reducers/auth';
import Menu from 'Components/Menu';
import User from 'utils/User';

const topNavMenuBtnClass = 'no-underline text-base-600 items-center cursor-pointer';

const MenuOptionComponent = ({ name, email, roles }) => (
    <div className="flex flex-col pl-2">
        <div className="font-700 normal-case" data-testid="menu-user-name">
            {name}
        </div>
        {email && (
            <div className="lowercase text-base-500 italic pt-px" data-testid="menu-user-email">
                {email}
            </div>
        )}
        <div className="pt-1" data-testid="menu-user-roles">
            <span className="font-700 pr-2">Roles ({roles.length}):</span>
            <span>{roles.map((role) => role.name).join(', ')}</span>
        </div>
    </div>
);

function TopNavBarMenu({ logout, userData }) {
    /**
     * TODO: rework the logic for the top-right menu
     * currently starts with the last item and we conditional unshift middle item,
     * then always unshift top item
     * The use of unshift is a bit odd, especially now that the UserMenu is no longer feature-flagged.
     * Even without that, this building up of array backwards is taking the whole UI-as-code idea too far.
     *
     * Menu component should probably be adapted to just take children
     */

    let buttonIcon = <MoreHorizontal className="mx-4 h-4 w-4 pointer-events-none" />;
    const buttonTextClassName = 'border rounded-full mx-3 p-3 text-xl border-base-400';

    const user = new User(userData);
    const { displayName } = user;
    let displayEmail = user.email;
    if (displayEmail === displayName) {
        displayEmail = null;
    }

    const options = [
        {
            component: (
                <MenuOptionComponent name={displayName} email={displayEmail} roles={user.roles} />
            ),
            link: '/main/user',
        },
        { label: 'Logout', onClick: () => logout() },
    ];
    buttonIcon = (
        <Avatar
            name={user.name || user.username}
            extraClassName="mx-3 h-10 w-10 flex items-center justify-center leading-none"
        />
    );

    return (
        <div className="flex items-center">
            <Menu
                className={`${topNavMenuBtnClass} h-full`}
                menuClassName="min-w-48"
                buttonIcon={buttonIcon}
                buttonTextClassName={buttonTextClassName}
                hideCaret
                options={options}
            />
        </div>
    );
}

TopNavBarMenu.propTypes = {
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

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(TopNavBarMenu));
