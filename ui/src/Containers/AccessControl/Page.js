import React from 'react';

import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import PageHeader from 'Components/PageHeader';
import Roles from 'Containers/AccessControl/Roles/Roles';
import AuthProviders from 'Containers/AccessControl/AuthProviders/AuthProviders';

function Page() {
    const tabHeaders = [
        { text: 'Auth Provider Rules', disabled: false },
        { text: 'Roles and Permissions', disabled: false }
    ];
    return (
        <section className="flex flex-col h-full">
            <PageHeader header="Access Control" />
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

Page.propTypes = {};

export default Page;
