import React from 'react';

import ListHeader from './Header';

const ComplianceListPage = () => (
    <section className="flex flex-col h-full">
        <ListHeader />
        <div className="flex-1 relative bg-base-200 p-4 overflow-auto" />
    </section>
);

export default ComplianceListPage;
