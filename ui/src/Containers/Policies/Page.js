import React from 'react';

import ConfirmationDialogue from 'Containers/Policies/ConfirmationDialogue';
import Header from 'Containers/Policies/Header';
import Table from 'Containers/Policies/Table/Table';
import Wizard from 'Containers/Policies/Wizard/Wizard';

// Top level policies page display in the APP frame.
export default function Page() {
    return (
        <section className="flex flex-1 flex-col h-full">
            <div>
                <Header />
            </div>
            <div className="flex flex-1 bg-base-200">
                <div className="flex w-full h-full bg-base-100 rounded-sm shadow">
                    <Table />
                    <Wizard />
                </div>
            </div>
            <ConfirmationDialogue />
        </section>
    );
}

Page.propTypes = {};
