import React, { useState } from 'react';
import { useLocation } from 'react-router-dom-v5-compat';

import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import PageHeader from 'Components/PageHeader';
import SidePanelAnimatedArea from 'Components/animations/SidePanelAnimatedArea';
import EntitiesMenu from 'Components/workflow/EntitiesMenu';
import getSidePanelEntity from 'utils/getSidePanelEntity';
import parseURL from 'utils/URLParser';
import workflowStateContext from 'Containers/workflowStateContext';
import { WorkflowState } from 'utils/WorkflowState';
import { getVulnerabilityManagementEntityTypes } from 'utils/entityRelationships';

import { entityNounSentenceCasePlural } from '../entitiesForVulnerabilityManagement';
import WorkflowSidePanel from '../WorkflowSidePanel';
import EntityComponent from '../Entity/VulnMgmtEntity';
import ListComponent from './VulnMgmtList';

const WorkflowListPageLayout = () => {
    const location = useLocation();

    const workflowState = parseURL(location);
    const { useCase, search, sort, paging } = workflowState;
    const pageState = new WorkflowState(
        useCase,
        workflowState.getPageStack(),
        search,
        sort,
        paging
    );

    // set up cache-busting system that either the list or sidepanel can use to trigger list refresh
    const [refreshTrigger, setRefreshTrigger] = useState(0);

    // Page props
    const pageListType = workflowState.getBaseEntity().entityType;
    const pageSearch = workflowState.search[searchParams.page];
    const pageSort = workflowState.sort[sortParams.page];
    const pagePaging = workflowState.paging[pagingParams.page];

    // Sidepanel props
    const { sidePanelEntityId, sidePanelEntityType, sidePanelListType } =
        getSidePanelEntity(workflowState);
    const sidePanelSearch = workflowState.search[searchParams.sidePanel];
    const sidePanelSort = workflowState.sort[sortParams.sidePanel];
    const sidePanelPaging = workflowState.paging[pagingParams.sidePanel];
    const selectedRow = workflowState.getSelectedTableRow();

    const header = entityNounSentenceCasePlural[pageListType];
    const entityContext = {};

    if (selectedRow) {
        const { entityType, entityId } = selectedRow;
        entityContext[entityType] = entityId;
    }

    return (
        <workflowStateContext.Provider value={pageState}>
            <div className="flex flex-col relative h-full">
                <PageHeader
                    header={header}
                    subHeader="Entity list"
                    classes="pr-0 ignore-react-onclickoutside"
                >
                    <div className="flex flex-1 justify-end h-full">
                        <div className="flex items-center pl-2">
                            <EntitiesMenu
                                text="All Entities"
                                options={getVulnerabilityManagementEntityTypes()}
                            />
                        </div>
                    </div>
                </PageHeader>
                <div className="h-full relative z-0 min-h-0 bg-base-100" id="capture-list">
                    <ListComponent
                        entityListType={pageListType}
                        selectedRowId={selectedRow && selectedRow.entityId}
                        search={pageSearch}
                        sort={pageSort}
                        page={pagePaging}
                        refreshTrigger={refreshTrigger}
                        setRefreshTrigger={setRefreshTrigger}
                    />
                    <SidePanelAnimatedArea isOpen={!!sidePanelEntityId}>
                        <WorkflowSidePanel>
                            <EntityComponent
                                entityId={sidePanelEntityId}
                                entityType={sidePanelEntityType}
                                entityListType={sidePanelListType}
                                search={sidePanelSearch}
                                sort={sidePanelSort}
                                page={sidePanelPaging}
                                entityContext={entityContext}
                                refreshTrigger={refreshTrigger}
                                setRefreshTrigger={setRefreshTrigger}
                            />
                        </WorkflowSidePanel>
                    </SidePanelAnimatedArea>
                </div>
            </div>
        </workflowStateContext.Provider>
    );
};

export default WorkflowListPageLayout;
