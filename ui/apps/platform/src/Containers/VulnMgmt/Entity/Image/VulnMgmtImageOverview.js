import React from 'react';
import pluralize from 'pluralize';
import cloneDeep from 'lodash/cloneDeep';
import { Alert, Button } from '@patternfly/react-core';

import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import LinkShim from 'Components/PatternFly/LinkShim';
import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import RiskScore from 'Components/RiskScore';
import TopCvssLabel from 'Components/TopCvssLabel';
import CVETable from 'Containers/Images/CVETable';
import { getWorkloadEntityPagePath } from 'Containers/Vulnerabilities/utils/searchUtils';
import ScanDataMessage from 'Containers/VulnMgmt/Components/ScanDataMessage';
import getImageScanMessage from 'Containers/VulnMgmt/VulnMgmt.utils/getImageScanMessage';
import TopRiskiestEntities from 'Containers/VulnMgmt/widgets/TopRiskiestEntities';
import CvesByCvssScore from 'Containers/VulnMgmt/widgets/CvesByCvssScore';
import entityTypes from 'constants/entityTypes';
import DateTimeField from 'Components/DateTimeField';
import { entityToColumns } from 'constants/listColumns';
import useFeatureFlags from 'hooks/useFeatureFlags';

import { entityGridContainerClassName } from '../WorkflowEntityPage';
import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidget from '../TableWidget';

const emptyImage = {
    deploymentCount: 0,
    id: '',
    lastUpdated: '',
    metadata: {
        layerShas: [],
        v1: {
            layers: [],
        },
    },
    name: {},
    priority: 0,
    scan: {
        components: [],
    },
    topVuln: {},
    vulnCount: 0,
};

const VulnMgmtImageOverview = ({ data, entityContext }) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isPlatformCveSplitEnabled = isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT');
    // guard against incomplete GraphQL-cached data
    const safeData = { ...emptyImage, ...data };
    const { metadata, scan, topVuln, priority, notes } = safeData;
    safeData.componentCount = scan?.components?.length || 0;

    // TODO: replace this hack with feature flag selection of components or imageComponents,
    //       after `layerIndex` is available on ImageComponent
    safeData.imageComponentCount = scan?.components?.length || 0;

    const layers = metadata ? cloneDeep(metadata.v1.layers) : [];
    const fixableCves = [];

    // If we have a scan, then we can try and assume we have layers
    if (scan) {
        layers.forEach((layer, i) => {
            layers[i].components = [];
            layers[i].cvesCount = 0;
        });
        scan.components.forEach((component) => {
            component.vulns.forEach((cve) => {
                if (cve.isFixable) {
                    fixableCves.push(cve);
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
            key: 'SHA',
            value: safeData.id,
        },
        {
            key: 'Created',
            value: (metadata?.v1 && <DateTimeField date={metadata.v1.created} asString />) || '-',
        },
        {
            key: 'Scanner',
            value: <span>{scan?.dataSource?.name || '-'}</span>,
        },
        {
            key: 'Scan time',
            value: (scan && <DateTimeField date={scan.scanTime} asString />) || '-',
        },
        {
            key: 'Image OS',
            value: <span>{scan?.operatingSystem || '-'}</span>,
        },
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

    const currentEntity = { [entityTypes.IMAGE]: data.id };
    const newEntityContext = { ...entityContext, ...currentEntity };

    const scanMessage = getImageScanMessage(notes || [], scan?.notes || []);

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow min-w-0">
                <ScanDataMessage header={scanMessage.header} body={scanMessage.body} />
                <CollapsibleSection title="Image Summary">
                    <div className={entityGridContainerClassName}>
                        <div className="s-1">
                            <Metadata
                                className="h-full sm:min-h-64 min-w-48 bg-base-100 pdf-page"
                                keyValuePairs={metadataKeyValuePairs}
                                statTiles={imageStats}
                                title="Details and metadata"
                            />
                        </div>
                        <div className="s-1">
                            <CvesByCvssScore entityContext={currentEntity} />
                        </div>
                        <div className="s-1">
                            <TopRiskiestEntities limit={5} entityContext={currentEntity} />
                        </div>
                    </div>
                </CollapsibleSection>
                <CollapsibleSection title="Dockerfile" defaultOpen={false}>
                    <div className="flex pdf-page pdf-stretch pdf-new rounded relative mb-4 ml-4 mr-4">
                        <TableWidget
                            header={`${layers.length} ${pluralize(
                                'layer',
                                layers.length
                            )} across this image`}
                            rows={layers}
                            entityType={entityTypes.IMAGE}
                            noDataText="No layers available in this image"
                            className="bg-base-100"
                            columns={entityToColumns[entityTypes.IMAGE]}
                            SubComponent={renderCVEsTable}
                            idAttribute="id"
                        />
                    </div>
                </CollapsibleSection>
                <CollapsibleSection id="image-findings" title="Image Findings" defaultOpen>
                    <div className="flex pdf-page pdf-stretch pdf-new rounded relative mb-4 ml-4 mr-4 pb-20">
                        <Alert
                            variant="warning"
                            isInline
                            title={
                                <>
                                    View this image in{' '}
                                    <Button
                                        component={LinkShim}
                                        variant="link"
                                        href={`${vulnerabilitiesWorkloadCvesPath}/${getWorkloadEntityPagePath('Image', data.id)}`}
                                        isInline
                                    >
                                        {isPlatformCveSplitEnabled
                                            ? 'User Workloads'
                                            : 'Workload CVEs'}
                                    </Button>{' '}
                                    for a detailed breakdown of detected vulnerabilities
                                </>
                            }
                            component="p"
                        />
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
