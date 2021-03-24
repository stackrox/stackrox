import React from 'react';

import { PageBody } from 'Components/Panel';
import Header from 'Containers/Policies/Header';
import Table from 'Containers/Policies/Table/Table';
import Wizard from 'Containers/Policies/Wizard/Wizard';
import PolicyBulkActionDialogue from 'Containers/Policies/PolicyBulkActionDialogue';

// Top level policies page display in the APP frame.
export default function Page() {
    return (
        <>
            <Header />
            <PageBody>
                <Table />
                <Wizard />
            </PageBody>
            <PolicyBulkActionDialogue />
        </>
    );
}

Page.propTypes = {};
