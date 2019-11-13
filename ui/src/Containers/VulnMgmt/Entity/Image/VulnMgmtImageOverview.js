import React, { useContext } from 'react';
import CollapsibleSection from 'Components/CollapsibleSection';

import Metadata from 'Components/Metadata';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import RiskScore from 'Components/RiskScore';
import TopCvssLabel from 'Components/TopCvssLabel';
import CVETable from 'Containers/Images/CVETable';
import workflowStateContext from 'Containers/workflowStateContext';
import entityTypes from 'constants/entityTypes';
import DateTimeField from 'Components/DateTimeField';
import { getCveTableColumns } from 'Containers/VulnMgmt/List/Cves/VulnMgmtListCves';
import { entityToColumns } from 'constants/listColumns';
import { resourceLabels } from 'messages/common';

import pluralize from 'pluralize';
import cloneDeep from 'lodash/cloneDeep';
import TopRiskiestImagesAndComponents from 'Containers/VulnMgmt/widgets/TopRiskiestImagesAndComponents';
import CvesByCvssScore from 'Containers/VulnMgmt/widgets/CvesByCvssScore';
import { entityGridContainerClassName } from 'Containers/Workflow/WorkflowEntityPage';

import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidget from '../TableWidget';

const emptyImage = {
    deploymentCount: 0,
    id: '',
    lastUpdated: '',
    metadata: {
        layerShas: [],
        v1: {}
    },
    name: {},
    priority: 0,
    scan: {},
    topVuln: {},
    vulnCount: 0
};

const VulnMgmtImageOverview = ({ data, entityContext }) => {
    const workflowState = useContext(workflowStateContext);

    // guard against incomplete GraphQL-cached data
    const safeData = { ...emptyImage, ...data };
    const { metadata, scan, topVuln, priority } = safeData;
    safeData.componentCount = scan && scan.components && scan.components.length;

    const layers = metadata ? cloneDeep(metadata.v1.layers) : [];
    const cves = [];

    // If we have a scan, then we can try and assume we have layers
    if (scan) {
        layers.forEach((layer, i) => {
            layers[i].components = [];
            layers[i].cvesCount = 0;
        });
        scan.components.forEach(component => {
            component.vulns.forEach(cve => {
                if (cve.isFixable) {
                    cves.push(cve);
                }
            });

            if (component.layerIndex !== undefined && layers[component.layerIndex]) {
                layers[component.layerIndex].components.push(component);
                layers[component.layerIndex].cvesCount += component.vulns.length;
            }
        });
    }

    const metadataKeyValuePairs = [
        {
            key: 'Created',
            value:
                (metadata && metadata.v1 && (
                    <DateTimeField date={metadata.v1.created} asString />
                )) ||
                'N/A'
        },
        {
            key: 'Scan time',
            value: (scan && <DateTimeField date={scan.scanTime} asString />) || 'N/A'
        }
    ];

    const imageStats = [<RiskScore key="risk-score" score={priority} />];
    if (topVuln) {
        const { cvss, scoreVersion } = topVuln;
        imageStats.push(
            <TopCvssLabel key="top-cvss" cvss={cvss} version={scoreVersion} expanded />
        );
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

    const newEntityContext = { ...entityContext, [entityTypes.IMAGE]: data.id };

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow min-w-0">
                <CollapsibleSection title="Image Summary">
                    <div className={entityGridContainerClassName}>
                        <div className="s-1">
                            <Metadata
                                className="h-full min-w-48 bg-base-100 bg-counts-widget"
                                keyValuePairs={metadataKeyValuePairs}
                                statTiles={imageStats}
                                title="Details & Metadata"
                            />
                        </div>
                        <div className="s-1">
                            <CvesByCvssScore entityContext={newEntityContext} />
                        </div>
                        <div className="s-1">
                            <TopRiskiestImagesAndComponents
                                limit={5}
                                entityContext={newEntityContext}
                            />
                        </div>
                    </div>
                </CollapsibleSection>
                <CollapsibleSection title="Image Findings">
                    <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                        <Tabs
                            hasTabSpacing
                            headers={[{ text: 'Fixable CVEs' }, { text: 'Dockerfile' }]}
                        >
                            <TabContent>
                                <TableWidget
                                    header={`${cves.length} fixable ${pluralize(
                                        resourceLabels.CVE,
                                        cves.length
                                    )} found across this image`}
                                    rows={cves}
                                    entityType={entityTypes.CVE}
                                    noDataText="No fixable CVEs available in this image"
                                    className="bg-base-100"
                                    columns={getCveTableColumns(workflowState)}
                                    idAttribute="cve"
                                />
                            </TabContent>
                            <TabContent>
                                <TableWidget
                                    header={`${layers.length} ${pluralize(
                                        'layer',
                                        layers.length
                                    )} layers across this image`}
                                    rows={layers}
                                    noDataText="No layers available in this image"
                                    className="bg-base-100"
                                    columns={entityToColumns[entityTypes.IMAGE]}
                                    SubComponent={renderCVEsTable}
                                    idAttribute="id"
                                />
                            </TabContent>
                        </Tabs>
                    </div>
                </CollapsibleSection>
            </div>
            <RelatedEntitiesSideList
                entityType={entityTypes.IMAGE}
                entityContext={newEntityContext}
                data={safeData}
            />
        </div>
    );
};

export default VulnMgmtImageOverview;
