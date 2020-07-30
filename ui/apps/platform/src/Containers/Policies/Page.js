import React from 'react';

import Header from 'Containers/Policies/Header';
import Table from 'Containers/Policies/Table/Table';
import Wizard from 'Containers/Policies/Wizard/Wizard';
import PolicyBulkActionDialogue from 'Containers/Policies/PolicyBulkActionDialogue';

// Top level policies page display in the APP frame.
export default function Page() {
    return (
        <section className="flex flex-1 flex-col h-full">
            <div>
                <Header />
            </div>
            <div className="flex flex-1">
                <div className="flex w-full h-full rounded-sm shadow">
                    <Table />
                    <Wizard />
                </div>
            </div>
            <PolicyBulkActionDialogue />
        </section>
    );
}

Page.propTypes = {};
