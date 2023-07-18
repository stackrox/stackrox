import React, { useContext } from 'react';

import CollapsibleSection from 'Components/CollapsibleSection';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import TopCvssLabel from 'Components/TopCvssLabel';
import RiskScore from 'Components/RiskScore';
import Metadata from 'Components/Metadata';
import CvesByCvssScore from 'Containers/VulnMgmt/widgets/CvesByCvssScore';
import { entityGridContainerBaseClassName } from '../WorkflowEntityPage';

import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidgetFixableCves from '../TableWidgetFixableCves';

const emptyComponent = {
    deploymentCount: 0,
    fixableCVEs: [],
    id: '',
    imageCount: 0,
    name: '',
    priority: 0,
    topVuln: {},
    version: '',
    vulnCount: 0,
    fixedIn: '',
};

function VulnMgmtComponentOverview({ data, entityContext }) {
    const workflowState = useContext(workflowStateContext);
    const currentEntityType = workflowState.getCurrentEntityType();

    const vulnType =
        currentEntityType === entityTypes.NODE_COMPONENT
            ? entityTypes.NODE_CVE
            : entityTypes.IMAGE_CVE;

    // guard against incomplete GraphQL-cached data
    const safeData = { ...emptyComponent, ...data };

    const { fixedIn, version, priority, topVuln, id, location, vulnCount, activeState } = safeData;
    const operatingSystem = safeData?.operatingSystem;

    const metadataKeyValuePairs = [
        {
            key: 'Component Version',
            value: version,
        },
    ];

    // check if this component is scoped to an image higher up in the hierarchy
    const hasImageAsAncestor = workflowState.getSingleAncestorOfType(entityTypes.IMAGE);
    // if scoped under an image, try to show component Location
    if (hasImageAsAncestor) {
        metadataKeyValuePairs.push({
            key: 'Location',
            value: location || 'N/A',
        });
    }

    if (operatingSystem !== undefined) {
        metadataKeyValuePairs.push({
            key: 'Operating System',
            value: operatingSystem,
        });
    }

    metadataKeyValuePairs.push({
        key: 'Fixed In',
        value: fixedIn || (vulnCount === 0 ? 'N/A' : 'Not Fixable'),
    });

    // check if this component is scoped to an image higher up in the hierarchy
    const hasDeploymentAsAncestor = workflowState.getSingleAncestorOfType(entityTypes.DEPLOYMENT);
    // if scoped under an image, try to show component Location
    if (hasDeploymentAsAncestor) {
        metadataKeyValuePairs.push({
            key: 'Active status',
            value: activeState?.state ?? 'Undetermined',
        });
    }

    const componentStats = [<RiskScore key="risk-score" score={priority} />];
    if (topVuln) {
        const { cvss, scoreVersion } = topVuln;
        componentStats.push(
            <TopCvssLabel key="top-cvss" cvss={cvss} version={scoreVersion} expanded />
        );
    }

    const currentEntity = { [currentEntityType]: id };
    const newEntityContext = { ...entityContext, ...currentEntity };

    const entityGridContainerClassName = `${entityGridContainerBaseClassName} grid-columns-1 md:grid-columns-2 lg:grid-columns-2`;

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow min-w-0">
                <CollapsibleSection title="Component Summary">
                    <div className={entityGridContainerClassName}>
                        <div className="s-1">
                            <Metadata
                                className="h-full min-w-48 bg-base-100 pdf-page"
                                keyValuePairs={metadataKeyValuePairs}
                                statTiles={componentStats}
                                title="Details and metadata"
                            />
                        </div>
                        <div className="s-1">
                            <CvesByCvssScore
                                entityContext={currentEntity}
                                parentContext={entityContext}
                            />
                        </div>
                    </div>
                </CollapsibleSection>
                <CollapsibleSection title="Component Findings">
                    <div className="flex pdf-page pdf-stretch shadow rounded relative bg-base-100 mb-4 ml-4 mr-4">
                        <TableWidgetFixableCves
                            workflowState={workflowState}
                            entityContext={entityContext}
                            entityType={currentEntityType}
                            vulnType={vulnType}
                            name={safeData?.name}
                            id={safeData?.id}
                        />
                    </div>
                </CollapsibleSection>
            </div>
            <RelatedEntitiesSideList
                entityType={currentEntityType}
                entityContext={newEntityContext}
                data={safeData}
            />
        </div>
    );
}

export default VulnMgmtComponentOverview;
