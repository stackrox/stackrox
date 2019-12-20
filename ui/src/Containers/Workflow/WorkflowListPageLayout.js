import React from 'react';
import pluralize from 'pluralize';
import startCase from 'lodash/startCase';

import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import PageHeader from 'Components/PageHeader';
import ExportButton from 'Components/ExportButton';
import EntitiesMenu from 'Components/workflow/EntitiesMenu';
import entityLabels from 'messages/entity';
import useCaseLabels from 'messages/useCase';
import getSidePanelEntity from 'utils/getSidePanelEntity';
import parseURL from 'modules/URLParser';
import workflowStateContext from 'Containers/workflowStateContext';
import { WorkflowState } from 'modules/WorkflowState';
import { useCaseEntityMap } from 'modules/entityRelationships';
import WorkflowSidePanel from './WorkflowSidePanel';
import { EntityComponentMap, ListComponentMap } from './UseCaseComponentMaps';

const WorkflowListPageLayout = ({ location }) => {
    const workflowState = parseURL(location);
    const { useCase, search, sort, paging } = workflowState;
    const pageState = new WorkflowState(
        useCase,
        workflowState.getPageStack(),
        search,
        sort,
        paging
    );

    // Get the list / entity components
    const ListComponent = ListComponentMap[useCase];
    const EntityComponent = EntityComponentMap[useCase];

    // Page props
    const pageListType = workflowState.getBaseEntity().entityType;
    const pageSearch = workflowState.search[searchParams.page];
    const pageSort = workflowState.sort[sortParams.page];
    const pagePaging = workflowState.paging[pagingParams.page];

    // Sidepanel props
    const { sidePanelEntityId, sidePanelEntityType, sidePanelListType } = getSidePanelEntity(
        workflowState
    );
    const sidePanelSearch = workflowState.search[searchParams.sidePanel];
    const sidePanelSort = workflowState.sort[sortParams.sidePanel];
    const sidePanelPaging = workflowState.paging[pagingParams.sidePanel];
    const selectedRow = workflowState.getSelectedTableRow();

    const header = pluralize(entityLabels[pageListType]);
    const exportFilename = `${useCaseLabels[useCase]} ${pluralize(startCase(header))} Report`;
    const entityContext = {};

    if (selectedRow) {
        const { entityType, entityId } = selectedRow;
        entityContext[entityType] = entityId;
    }

    return (
        <workflowStateContext.Provider value={pageState}>
            <div className="flex flex-col relative min-h-full">
                <PageHeader header={header} subHeader="Entity List" classes="pr-0">
                    <div className="flex flex-1 justify-end h-full">
                        <div className="flex items-center pr-2">
                            <ExportButton
                                fileName={exportFilename}
                                type={pageListType}
                                page={useCase}
                                disabled={!!sidePanelEntityId}
                                pdfId="capture-list"
                            />
                        </div>
                        <div className="flex items-center pl-2">
                            <EntitiesMenu
                                text="All Entities"
                                options={useCaseEntityMap[useCase]}
                                grouped
                            />
                        </div>
                    </div>
                </PageHeader>
                <div className="h-full bg-base-100 relative z-0" id="capture-list">
                    <ListComponent
                        entityListType={pageListType}
                        selectedRowId={selectedRow && selectedRow.entityId}
                        search={pageSearch}
                        sort={pageSort}
                        page={pagePaging}
                    />
                    <WorkflowSidePanel isOpen={!!sidePanelEntityId}>
                        {sidePanelEntityId ? (
                            <EntityComponent
                                entityId={sidePanelEntityId}
                                entityType={sidePanelEntityType}
                                entityListType={sidePanelListType}
                                search={sidePanelSearch}
                                sort={sidePanelSort}
                                page={sidePanelPaging}
                                entityContext={entityContext}
                            />
                        ) : (
                            <span />
                        )}
                    </WorkflowSidePanel>
                </div>
            </div>
        </workflowStateContext.Provider>
    );
};

export default WorkflowListPageLayout;
