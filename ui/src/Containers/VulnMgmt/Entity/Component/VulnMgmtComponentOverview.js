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
import { entityGridContainerClassName } from 'Containers/Workflow/WorkflowEntityPage';

import TableWidget from '../TableWidget';
import RelatedEntitiesSideList from '../RelatedEntitiesSideList';

const emptyComponent = {
    deploymentCount: 0,
    fixableCVEs: [],
    id: '',
    imageCount: 0,
    name: '',
    priority: 0,
    topVuln: {},
    version: '',
    vulnCount: 0
};

function VulnMgmtComponentOverview({ data, entityContext }) {
    const workflowState = useContext(workflowStateContext);

    // guard against incomplete GraphQL-cached data
    const safeData = { ...emptyComponent, ...data };

    const { version, priority, topVuln, fixableCVEs, id } = safeData;

    const metadataKeyValuePairs = [
        {
            key: 'Component Version',
            value: version
        }
    ];

    const componentStats = [<RiskScore key="risk-score" score={priority} />];
    if (topVuln) {
        const { cvss, scoreVersion } = topVuln;
        componentStats.push(
            <TopCvssLabel key="top-cvss" cvss={cvss} version={scoreVersion} expanded />
        );
    }

    const newEntityContext = { ...entityContext, [entityTypes.COMPONENT]: id };

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow min-w-0">
                <CollapsibleSection title="Component Summary" />
                <div className={entityGridContainerClassName}>
                    <div className="s-1">
                        <Metadata
                            className="h-full min-w-48 bg-base-100 bg-counts-widget"
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
                            )} found across this component`}
                            rows={fixableCVEs}
                            entityType={entityTypes.CVE}
                            noDataText="No fixable CVEs available in this component"
                            className="bg-base-100"
                            columns={getCveTableColumns(workflowState)}
                            idAttribute="cve"
                        />
                    </div>
                </CollapsibleSection>
            </div>
            <RelatedEntitiesSideList
                entityType={entityTypes.COMPONENT}
                entityContext={newEntityContext}
                data={safeData}
            />
        </div>
    );
}

export default VulnMgmtComponentOverview;
