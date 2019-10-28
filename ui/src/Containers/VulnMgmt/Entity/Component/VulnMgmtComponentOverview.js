import React, { useContext } from 'react';

import CollapsibleSection from 'Components/CollapsibleSection';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import TopCvssLabel from 'Components/TopCvssLabel';
import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';
// import TableWidget from 'Containers/ConfigManagement/Entity/widgets/TableWidget';
import {
    getCveTableColumns,
    renderCveDescription,
    defaultCveSort
} from 'Containers/VulnMgmt/List/Cves/VulnMgmtListCves';
import EntityList from 'Components/EntityList';

// import RelatedEntitiesSideList from '../RelatedEntitiesSideList';

function VulnMgmtComponentOverview({ data }) {
    const workflowState = useContext(workflowStateContext);

    const { version, priority, vulns } = data;

    const topVuln = vulns.reduce((max, curr) => (curr.cvss > max.cvss ? curr : max), {
        cvss: 0,
        scoreVersion: null
    });
    const { cvss, scoreVersion } = topVuln;

    // Expand all rows to include description
    function getDefaultExpandedRows(items) {
        return items.map((_element, index) => {
            return { [index]: true };
        });
    }

    const cveTableColumns = getCveTableColumns(workflowState).filter(
        col => !['Deployments', 'Images', 'Components'].includes(col.Header)
    );
    const expandedCveRows = getDefaultExpandedRows(vulns);
    return (
        <div className="w-full h-full" id="capture-dashboard-stretch">
            <div className="flex h-full">
                <div className="flex flex-col flex-grow">
                    <CollapsibleSection title="Component Summary" />
                    <div className="flex mb-4 pdf-page">
                        <Widget
                            header="Details & Metadata"
                            className="mx-4 bg-base-100 h-48 mb-4 bg-counts-widget flex-grow"
                        >
                            <div className="flex flex-col w-full">
                                <div className="flex border-b border-base-400">
                                    <div className="flex flex-grow p-4 justify-center items-center border-r-2 border-base-300 border-dotted">
                                        <span className="pr-1">Risk score:</span>
                                        <span className="pl-1 text-3xl">{priority}</span>
                                    </div>
                                    <div className="flex flex-col p-4 flex-grow justify-center text-center">
                                        <TopCvssLabel cvss={cvss} version={scoreVersion} expanded />
                                    </div>
                                </div>
                                <div className="flex flex-col border-base-400 pl-2 pr-2">
                                    <div className="flex p-3 border-b border-base-400">
                                        <span className="text-base-700 font-600 mr-2">
                                            Component Version:
                                        </span>
                                        {version}
                                    </div>
                                </div>
                            </div>
                        </Widget>
                        <Widget header="CVEs By CVSS Score" className="mx-4 bg-base-100 h-48 mb-4 ">
                            <div>hi</div>
                        </Widget>
                        <Widget
                            header="Top Riskiest Components"
                            className="mx-4 bg-base-100 h-48 mb-4 "
                        >
                            <div>hi</div>
                        </Widget>
                    </div>
                    <CollapsibleSection title="Component Findings">
                        <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                            {vulns.length === 0 ? (
                                <NoResultsMessage
                                    message="No fixable CVEs found across this component"
                                    className="p-6"
                                />
                            ) : (
                                <EntityList
                                    entityType={entityTypes.CVE}
                                    idAttribute="cve"
                                    rowData={vulns}
                                    tableColumns={cveTableColumns}
                                    selectedRowId={null}
                                    search={null}
                                    SubComponent={renderCveDescription}
                                    defaultSorted={defaultCveSort}
                                    defaultExpanded={expandedCveRows}
                                />
                                // <TableWidget
                                //     header={`${
                                //         vulns.length
                                //     } fixable CVEs have been found across this component`}
                                //     rows={vulns}
                                //     noDataText="No fixable CVEs"
                                //     className="bg-base-100"
                                //     columns={entityToColumns[entityTypes.CVE]}
                                //     // SubComponent={renderCVEsTable}
                                //     idAttribute="id"
                                // />
                            )}
                        </div>
                    </CollapsibleSection>
                </div>
                {/* Tabs are dynamic. We shouldn't have to define a count function for every related entity type we expect */}
                {/* <RelatedEntitiesSideList
                    entityType={entityTypes.IMAGE}
                    workflowState={workflowState}
                    getCountData={getCountData}
                /> */}
            </div>
        </div>
    );
}

export default VulnMgmtComponentOverview;
