import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { ArrowRightCircle } from 'react-feather';

import { selectors } from 'reducers';
import PageHeader from 'Components/PageHeader';
import User from 'utils/User';
import SideBar from 'Containers/AccessControl/SideBar';
import Permissions from 'Containers/AccessControl/Roles/Permissions/Permissions';

const UserPage = ({ userData }) => {
    const user = new User(userData);
    const authProviderName =
        user.usedAuthProvider?.type === 'basic' ? 'Basic' : user.usedAuthProvider?.name;
    const aggregatedPermissionsPage = {
        username: user.username || 'Unknown',
        authProviderName,
        resourceToAccess: user.resourceToAccessByRole,
    };
    const [selectedPage, setSelectedPage] = useState(aggregatedPermissionsPage);
    function onAggregatedPermissionsClick() {
        setSelectedPage(aggregatedPermissionsPage);
    }

    return (
        <section className="flex flex-1 h-full w-full">
            <div className="flex flex-1 flex-col w-full">
                <PageHeader
                    header={user.name || user.username}
                    subHeader={user.email}
                    capitalize={false}
                />
                <div className="flex bg-base-200">
                    <div className="m-4 shadow-sm w-1/4">
                        <button
                            type="button"
                            onClick={onAggregatedPermissionsClick}
                            className={`flex w-full h-14 border pl-4 pr-3 justify-between mb-4 text-base-600 items-center tracking-wide leading-normal font-700 uppercase ${
                                selectedPage === aggregatedPermissionsPage
                                    ? 'border-tertiary-400 bg-tertiary-200'
                                    : 'hover:bg-base-200 bg-base-100 border-base-400'
                            }`}
                        >
                            StackRox User Permissions
                            <ArrowRightCircle className="w-5" />
                        </button>
                        <SideBar
                            header="StackRox User Roles"
                            rows={user.roles}
                            selected={selectedPage}
                            onSelectRow={setSelectedPage}
                            type="role"
                            short
                        />
                    </div>
                    <div className="md:w-3/4 w-full my-4 mr-4">
                        <Permissions
                            resources={selectedPage.resourceToAccess}
                            selectedRole={selectedPage}
                            readOnly
                        />
                    </div>
                </div>
            </div>
        </section>
    );
};

UserPage.propTypes = {
    userData: PropTypes.shape({
        userAttributes: PropTypes.arrayOf(PropTypes.shape({})),
        userInfo: PropTypes.shape({
            roles: PropTypes.arrayOf(PropTypes.shape({})),
            permissions: PropTypes.shape({}),
        }),
    }).isRequired,
};

const mapStateToProps = createStructuredSelector({
    userData: selectors.getCurrentUser,
});

export default connect(mapStateToProps, null)(UserPage);
