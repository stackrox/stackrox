import React, { useContext } from 'react';
import pluralize from 'pluralize';

import CollapsibleSection from 'Components/CollapsibleSection';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import TopCvssLabel from 'Components/TopCvssLabel';
import RiskScore from 'Components/RiskScore';
import Metadata from 'Components/Metadata';
import CvesByCvssScore from 'Containers/VulnMgmt/widgets/CvesByCvssScore';
import { getCveTableColumns } from 'Containers/VulnMgmt/List/Cves/VulnMgmtListCves';

import TableWidget from '../TableWidget';
import RelatedEntitiesSideList from '../RelatedEntitiesSideList';

function VulnMgmtComponentOverview({ data, entityContext }) {
    const workflowState = useContext(workflowStateContext);

    const { version, priority, vulns, id, vulnCount, deploymentCount } = data;
    const fixableCVEs = vulns.filter(vuln => vuln.isFixable);

    const topVuln = vulns.reduce((max, curr) => (curr.cvss > max.cvss ? curr : max), {
        cvss: 0,
        scoreVersion: null
    });
    const { cvss, scoreVersion } = topVuln;

    const metadataKeyValuePairs = [
        {
            key: 'Component Version',
            value: version
        }
    ];

    const componentStats = [
        <RiskScore key="risk-score" score={priority} />,
        <TopCvssLabel key="top-cvss" cvss={cvss} version={scoreVersion} expanded />
    ];

    function getCountData(entityType) {
        switch (entityType) {
            case entityTypes.DEPLOYMENT:
                return deploymentCount;
            case entityTypes.CVE:
                return vulnCount;
            default:
                return 0;
        }
    }

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow">
                <CollapsibleSection title="Component Summary" />
                <div className="mx-4 grid grid-gap-6 xxxl:grid-gap-8 md:grid-columns-3 mb-4 pdf-page">
                    <div className="s-1">
                        <Metadata
                            className="h-full min-w-48 bg-base-100"
                            keyValuePairs={metadataKeyValuePairs}
                            statTiles={componentStats}
                            title="Details & Metadata"
                        />
                    </div>
                    <div className="s-1">
                        <CvesByCvssScore
                            entityContext={{ ...entityContext, [entityTypes.COMPONENT]: id }}
                        />
                    </div>
                </div>
                <CollapsibleSection title="Component Findings">
                    <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                        <TableWidget
                            header={`${fixableCVEs.length} fixable ${pluralize(
                                entityTypes.CVE,
                                fixableCVEs.length
                            )} found across this image`}
                            rows={fixableCVEs}
                            entityType={entityTypes.CVE}
                            noDataText="No fixable CVEs available in this component"
                            className="bg-base-100"
                            columns={getCveTableColumns(workflowState, false)}
                            idAttribute="cve"
                        />
                    </div>
                </CollapsibleSection>
            </div>
            {/* TO DO: Tabs are dynamic. We shouldn't have to define a count function for every related entity type we expect */}
            <RelatedEntitiesSideList
                entityType={entityTypes.COMPONENT}
                workflowState={workflowState}
                getCountData={getCountData}
            />
        </div>
    );
}

export default VulnMgmtComponentOverview;
