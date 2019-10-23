import React, { useContext } from 'react';
import CollapsibleSection from 'Components/CollapsibleSection';

import Widget from 'Components/Widget';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import NoResultsMessage from 'Components/NoResultsMessage';
import TopCvssLabel from 'Components/TopCvssLabel';
import TableWidget from 'Containers/ConfigManagement/Entity/widgets/TableWidget';
import CVETable from 'Containers/Images/CVETable';
import workflowStateContext from 'Containers/workflowStateContext';
import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import { entityToColumns } from 'constants/listColumns';

import { format } from 'date-fns';
import cloneDeep from 'lodash/cloneDeep';

import RelatedEntitiesSideList from '../RelatedEntitiesSideList';

const VulnMgmtImageOverview = ({ data }) => {
    const workflowState = useContext(workflowStateContext);

    const { metadata, scan, topVuln, deploymentCount, vulnCounter } = data;
    const { cvss, scoreVersion } = topVuln;

    const layers = metadata ? cloneDeep(metadata.v1.layers) : [];

    // If we have a scan, then we can try and assume we have layers
    if (scan) {
        layers.forEach((layer, i) => {
            layers[i].components = [];
        });
        scan.components.forEach(component => {
            if (component.layerIndex !== undefined && layers[component.layerIndex]) {
                layers[component.layerIndex].components.push(component);
            }
        });

        layers.forEach((layer, i) => {
            layers[i].cvesCount = layer.components.reduce((cnt, o) => cnt + o.vulns.length, 0);
        });
    }

    function renderCVEsTable(row) {
        const layer = row.original;
        if (!layer.components || layer.components.length === 0) {
            return null;
        }
        return (
            <CVETable
                scan={layer}
                containsFixableCVEs={false}
                className="cve-table my-3 ml-4 px-2 border-0 border-l-4 border-base-300"
            />
        );
    }

    function getCountData(entityType) {
        switch (entityType) {
            case entityTypes.DEPLOYMENT:
                return deploymentCount;
            case entityTypes.COMPONENT:
                if (scan && scan.components) return scan.components.length;
                return 0;
            case entityTypes.CVE:
                return vulnCounter.all.total;
            default:
                return 0;
        }
    }

    return (
        <div className="w-full h-full" id="capture-dashboard-stretch">
            <div className="flex h-full">
                <div className="flex flex-col flex-grow">
                    <CollapsibleSection title="Image Summary">
                        <div className="flex mb-4 pdf-page">
                            <Widget
                                header="Details & Metadata"
                                className="mx-4 bg-base-100 h-48 mb-4 bg-counts-widget flex-grow"
                            >
                                <div className="flex flex-col w-full">
                                    <div className="flex border-b border-base-400">
                                        <div className="p-4 border-r border-dashed border-base-400">
                                            {/* <span className="pr-1 font-weight-600">Risk:</span>
                                        <span className="text-xl">3</span> */}
                                        </div>
                                        <div className="flex flex-col p-4">
                                            <TopCvssLabel
                                                cvss={cvss}
                                                version={scoreVersion}
                                                expanded
                                            />
                                        </div>
                                    </div>
                                    <div className="flex flex-col border-base-400 p-4">
                                        <div className="flex p-3 border-b border-base-400">
                                            <span className="text-base-700 font-600 mr-2">
                                                Created:
                                            </span>
                                            {(metadata &&
                                                metadata.v1 &&
                                                format(metadata.v1.created, dateTimeFormat)) ||
                                                'N/A'}
                                        </div>
                                        <div className="flex p-3 border-b border-base-400">
                                            <span className="text-base-700 font-600 mr-2">
                                                Scan time:
                                            </span>
                                            {(scan && format(scan.scanTime, dateTimeFormat)) ||
                                                'N/A'}
                                        </div>
                                    </div>
                                </div>
                            </Widget>
                            <Widget
                                header="CVEs By CVSS Score"
                                className="mx-4 bg-base-100 h-48 mb-4 "
                            >
                                <div>hi</div>
                            </Widget>
                            <Widget
                                header="Top Riskiest Components"
                                className="mx-4 bg-base-100 h-48 mb-4 "
                            >
                                <div>hi</div>
                            </Widget>
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Image Findings">
                        <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                            <Tabs
                                hasTabSpacing
                                headers={[{ text: 'Dockerfile' }, { text: 'CVEs' }]}
                            >
                                <TabContent>
                                    {layers.length === 0 && (
                                        <NoResultsMessage
                                            message="No layers available in this image"
                                            className="p-6"
                                        />
                                    )}
                                    {layers.length > 0 && (
                                        <TableWidget
                                            header={`${layers.length} layers across this image`}
                                            rows={layers}
                                            noDataText="No Layers"
                                            className="bg-base-100"
                                            columns={entityToColumns[entityTypes.IMAGE]}
                                            SubComponent={renderCVEsTable}
                                            idAttribute="id"
                                        />
                                    )}
                                </TabContent>
                                <TabContent>
                                    <div>hello cves</div>
                                </TabContent>
                            </Tabs>
                        </div>
                    </CollapsibleSection>
                </div>
                <RelatedEntitiesSideList
                    entityType={entityTypes.IMAGE}
                    workflowState={workflowState}
                    getCountData={getCountData}
                />
            </div>
        </div>
    );
};

export default VulnMgmtImageOverview;
