import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import PageHeader from 'Components/PageHeader';
import getUserAttributeMap from 'utils/userDataUtils';
import SideBar from 'Containers/AccessControl/SideBar';
import Permissions from 'Containers/AccessControl/Roles/Permissions/Permissions';

const UserPage = ({ userData }) => {
    const [selectedPage, setSelectedPage] = useState(userData.userInfo.roles[0]);
    const { userAttributes, userInfo } = userData;
    const userAttributeMap = getUserAttributeMap(userAttributes);
    const header = userAttributeMap.name;
    const subHeader = userAttributeMap.email;
    return (
        <section className="flex flex-1 h-full w-full">
            <div className="flex flex-1 flex-col w-full">
                <PageHeader header={header} subHeader={subHeader} capitalize={false} />
                <div className="flex bg-base-200">
                    <div className="m-4 shadow-sm w-1/4">
                        <SideBar
                            header="StackRox User Roles"
                            rows={userInfo.roles}
                            selected={selectedPage}
                            onSelectRow={setSelectedPage}
                            type="role"
                        />
                    </div>
                    <div className="md:w-3/4 w-full my-4 mr-4">
                        <Permissions
                            isEditing={false}
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
        }),
    }).isRequired,
};

const mapStateToProps = createStructuredSelector({
    userData: selectors.getCurrentUser,
});

export default connect(mapStateToProps, null)(UserPage);
