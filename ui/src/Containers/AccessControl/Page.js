import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';

import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import PageHeader from 'Components/PageHeader';
import Roles from 'Containers/AccessControl/Roles/Roles';
import AuthProviders from 'Containers/AccessControl/AuthProviders/AuthProviders';
import { actions } from 'reducers/roles';

function Page({ fetchResources }) {
    fetchResources();
    const tabHeaders = [
        { text: 'Auth Provider Rules', disabled: false },
        { text: 'Roles and Permissions', disabled: false }
    ];
    return (
        <section className="flex flex-col h-full">
            <div className="flex flex-no-shrink">
                <PageHeader header="Access Control" />
            </div>
            <div className="flex h-full">
                <Tabs headers={tabHeaders}>
                    <TabContent>
                        <AuthProviders />
                    </TabContent>
                    <TabContent>
                        <Roles />
                    </TabContent>
                </Tabs>
            </div>
        </section>
    );
}

Page.propTypes = {
    fetchResources: PropTypes.func.isRequired
};

const mapDispatchToProps = {
    fetchResources: actions.fetchResources.request
};

export default connect(
    null,
    mapDispatchToProps
)(Page);
