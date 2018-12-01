import React from 'react';

import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import PageHeader from 'Components/PageHeader';
import Roles from 'Containers/AccessControl/Roles/Roles';
import IdentityProviders from 'Containers/AccessControl/IdentityProviders/IdentityProviders';

function Page() {
    const tabHeaders = [
        { text: 'Roles and Permissions', disabled: false },
        { text: 'Identity Provider Rules', disabled: false }
    ];
    return (
        <section className="flex flex-col h-full">
            <PageHeader header="Access Control" />
            <div className="flex h-full">
                <Tabs headers={tabHeaders}>
                    <TabContent>
                        <Roles />
                    </TabContent>
                    <TabContent>
                        <IdentityProviders />
                    </TabContent>
                </Tabs>
            </div>
        </section>
    );
}

Page.propTypes = {};

export default Page;
