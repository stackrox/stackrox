import React, { useContext } from 'react';
import pluralize from 'pluralize';
import startCase from 'lodash/startCase';
import searchContexts from 'constants/searchContexts';
import PageHeader from 'Components/PageHeader';
import ExportButton from 'Components/ExportButton';

import entityLabels from 'messages/entity';
import getSidePanelEntity from 'utils/getSidePanelEntity';
import { parseURL } from 'modules/URLReadWrite';

import WorkflowSidePanel from './WorkflowSidePanel';
import workflowStateContext from '../workflowStateContext';

import { EntityComponentMap, ListComponentMap } from './UseCaseComponentMaps';

const WorkflowListPageLayout = ({ location }) => {
    const workflowState = useContext(workflowStateContext);
    const { stateStack, useCase } = workflowState;
    const { searchState } = parseURL(location);
    const pageSearch = searchState[searchContexts.page];

    // Get the list / entity components
    const ListComponent = ListComponentMap[useCase];
    const EntityComponent = EntityComponentMap[useCase];

    // Calculate page entity props
    const pageListType = stateStack[0].entityType;

    // Calculate sidepanel entity props
    const {
        sidePanelEntityId,
        sidePanelEntityType,
        sidePanelListType,
        sidePanelSearch
    } = getSidePanelEntity(stateStack, searchState);

    const header = pluralize(entityLabels[pageListType]);
    const exportFilename = `${pluralize(startCase(header))} Report`;

    return (
        <div className="flex flex-col relative min-h-full">
            <PageHeader header={header} subHeader="Entity List">
                <div className="flex flex-1 justify-end">
                    <div className="flex">
                        <div className="flex items-center">
                            <ExportButton
                                fileName={exportFilename}
                                type={pageListType}
                                page="configManagement"
                                pdfId="capture-list"
                            />
                        </div>
                    </div>
                </div>
            </PageHeader>
            <div className="flex flex-1 h-full bg-base-100 relative z-0" id="capture-list">
                <ListComponent
                    // wrapperClass={`bg-base-100 ${sidePanelEntityId ? 'overlay' : ''}`}
                    entityListType={pageListType}
                    entityId={sidePanelEntityId}
                    search={pageSearch}
                />
            </div>
            <WorkflowSidePanel isOpen={!!sidePanelEntityId}>
                {sidePanelEntityId ? (
                    <EntityComponent
                        entityId={sidePanelEntityId}
                        entityType={sidePanelEntityType}
                        listType={sidePanelListType}
                        search={sidePanelSearch}
                    />
                ) : (
                    <span />
                )}
            </WorkflowSidePanel>
        </div>
    );
};

export default WorkflowListPageLayout;
