import React, { useState } from 'react';

import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import { resourceTypes } from 'constants/entityTypes';
import StandardsByEntity from 'Containers/Compliance/widgets/StandardsByEntity';
import StandardsAcrossEntity from 'Containers/Compliance/widgets/StandardsAcrossEntity';
import ComplianceByStandards from 'Containers/Compliance/widgets/ComplianceByStandards';
import DashboardHeader from './Header';

const ComplianceDashboardPage = () => {
    const [isExporting, setIsExporting] = useState(false);
    return (
        <section>
            <DashboardHeader
                classes="z-10 sticky top-0"
                isExporting={isExporting}
                setIsExporting={setIsExporting}
            />

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
                    <ComplianceByStandards />
                </div>
            </div>
            {isExporting && <BackdropExporting />}
        </section>
    );
};
export default ComplianceDashboardPage;
