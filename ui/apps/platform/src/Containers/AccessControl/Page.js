import React, { useEffect } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { createStructuredSelector } from 'reselect';

import { Message } from '@stackrox/ui-components';
import Tabs from 'Components/Tabs';
import Tab from 'Components/Tab';
import PageHeader from 'Components/PageHeader';
import Roles from 'Containers/AccessControl/Roles/Roles';
import AuthProviders from 'Containers/AccessControl/AuthProviders/AuthProviders';
import { actions } from 'reducers/roles';
import { selectors } from 'reducers';

function Page({ userRolePermissions, fetchResources }) {
    useEffect(() => {
        fetchResources();
    }, [fetchResources]);
    const tabHeaders = [
        { text: 'Auth Provider Rules', disabled: false },
        { text: 'Roles and Permissions', disabled: false },
    ];
    if (
        !userRolePermissions?.resourceToAccess?.AuthProvider ||
        userRolePermissions.resourceToAccess.AuthProvider === 'NO_ACCESS'
    ) {
        return (
            <div className="m-4">
                <Message type="error">You do not have permission to view Access Control.</Message>
            </div>
        );
    }

    return (
        <section className="flex flex-col h-full">
            <div className="flex flex-shrink-0">
                <PageHeader header="Access Control" />
            </div>
            <div className="flex h-full flex-1">
                <Tabs headers={tabHeaders}>
                    <Tab>
                        <AuthProviders />
                    </Tab>
                    <Tab>
                        <Roles />
                    </Tab>
                </Tabs>
            </div>
        </section>
    );
}

Page.propTypes = {
    userRolePermissions: PropTypes.shape({
        resourceToAccess: PropTypes.shape({ AuthProvider: PropTypes.string }),
    }).isRequired,
    fetchResources: PropTypes.func.isRequired,
};
const mapStateToProps = createStructuredSelector({
    userRolePermissions: selectors.getUserRolePermissions,
});

const mapDispatchToProps = {
    fetchResources: actions.fetchResources.request,
};

export default connect(mapStateToProps, mapDispatchToProps)(Page);
