import React from 'react';
import ComplianceByStandard from 'Containers/Compliance2/widgets/ComplianceByStandard';
import RelatedEntitiesList from 'Containers/Compliance2/widgets/RelatedEntitiesList';
import Header from './Header';

const ComplianceEntityPage = () => (
    <section className="flex flex-col h-full">
        <Header />
        <div className="flex-1 relative bg-base-200 p-4 overflow-auto">
            <div className="grid xl:grid-columns-3 md:grid-columns-2 sm:grid-columns-1 grid-gap-6">
                <ComplianceByStandard standard="PCI" />
                <ComplianceByStandard standard="NIST" />
                <ComplianceByStandard standard="HIPAA" />
                <ComplianceByStandard standard="CIS" />
                <RelatedEntitiesList />
            </div>
        </div>
    </section>
);

export default ComplianceEntityPage;
