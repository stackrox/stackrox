import React from 'react';
import { resourceTypes, standardTypes } from 'constants/entityTypes';

import StandardsByEntity from 'Containers/Compliance/widgets/StandardsByEntity';
import StandardsAcrossEntity from 'Containers/Compliance/widgets/StandardsAcrossEntity';
import ComplianceByStandard from 'Containers/Compliance/widgets/ComplianceByStandard';
import DashboardHeader from './Header';

const ComplianceDashboardPage = () => {
    return (
        <section>
            <DashboardHeader classes="z-10 sticky top-0" />

            <div className="flex-1 relative p-6 xxxl:p-8 bg-base-200" id="capture-dashboard">
                <div
                    className="grid grid-gap-6 xxxl:grid-gap-8 md:grid-auto-fit xxl:grid-auto-fit-wide md:grid-dense"
                    style={{ '--min-tile-height': '160px' }}
                >
                    <StandardsAcrossEntity
                        entityType={resourceTypes.CLUSTER}
                        bodyClassName="pr-4 py-1"
                        className="pdf-page"
                    />
                    <StandardsByEntity
                        entityType={resourceTypes.CLUSTER}
                        bodyClassName="p-4"
                        className="pdf-page"
                    />
                    <StandardsAcrossEntity
                        entityType={resourceTypes.NAMESPACE}
                        bodyClassName="px-4 pt-1"
                        className="pdf-page"
                    />
                    <StandardsAcrossEntity
                        entityType={resourceTypes.NODE}
                        bodyClassName="pr-4 py-1"
                        className="pdf-page"
                    />
                    <ComplianceByStandard
                        standardType={standardTypes.CIS_Docker_v1_2_0}
                        className="pdf-page"
                    />
                    <ComplianceByStandard
                        standardType={standardTypes.CIS_Kubernetes_v1_5}
                        className="pdf-page"
                    />
                    <ComplianceByStandard
                        standardType={standardTypes.HIPAA_164}
                        className="pdf-page"
                    />
                    <ComplianceByStandard
                        standardType={standardTypes.NIST_800_190}
                        className="pdf-page"
                    />
                    <ComplianceByStandard
                        standardType={standardTypes.NIST_SP_800_53_Rev_4}
                        className="pdf-page"
                    />
                    <ComplianceByStandard
                        standardType={standardTypes.PCI_DSS_3_2}
                        className="pdf-page"
                    />
                </div>
            </div>
        </section>
    );
};
export default ComplianceDashboardPage;
