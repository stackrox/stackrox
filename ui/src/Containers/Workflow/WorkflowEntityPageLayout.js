import React, { useContext, useEffect, useState } from 'react';
import { withRouter } from 'react-router-dom';
import searchContext from 'Containers/searchContext';
import PageHeader from 'Components/PageHeader';
import useCaseTypes from 'constants/useCaseTypes';
import VulnMgmtEntity from 'Containers/VulnMgmt/Entity/VulnMgmtEntity';
import VulnMgmtList from 'Containers/VulnMgmt/List/VulnMgmtList';
import EntityTabs from 'Components/EntityTabs';
import workflowStateContext from 'Containers/workflowStateContext';
import SidePanelAnimation from 'Components/animations/SidePanelAnimation';
import searchContexts from 'constants/searchContexts';
import { parseURL } from 'modules/URLReadWrite';
import WorkflowSidePanel from './WorkflowSidePanel';

// TODO: extract these map objects to somewhere common and reusable
const EntityComponentMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtEntity
};

// const EntityQueryMap = {
//     [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtEntityQueries
// };

// TODO: build out lists where needed in the sidebar
// eslint-disable-next-line no-unused-vars
const ListMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtList
};

const WorkflowEntityPageLayout = ({ location }) => {
    const workflowState = useContext(workflowStateContext);
    const { stateStack, useCase } = workflowState;
    const { searchState } = parseURL(location);
    const pageSearch = searchState[searchContexts.page];
    const sidePanelSearch = searchState[searchContexts.sidePanel];
    const EntityComponent = EntityComponentMap[useCase];

    // Calculate page entity props
    const pageEntity = stateStack[0];
    const { entityId: pageEntityId, entityType: pageEntityType } = pageEntity;
    const pageListType = stateStack[1] && stateStack[1].entityType;
    // Calculate sidepanel entity props
    const sidePanelStateStack = [...stateStack.slice(2)];
    const topItem = sidePanelStateStack.pop();
    const secondItem = sidePanelStateStack.pop();
    const sidePanelOpen = !!topItem;

    let sidePanelEntityId;
    let sidePanelEntityType;
    let sidePanelListType;
    if (sidePanelOpen) {
        if (topItem.entityId) {
            sidePanelEntityId = topItem.entityId;
            sidePanelEntityType = topItem.entityType;
        } else {
            sidePanelEntityId = secondItem.entityId;
            sidePanelEntityType = secondItem.entityType;
            sidePanelListType = topItem.entityType;
        }
    }

    const [fadeIn, setFadeIn] = useState(false);
    useEffect(() => setFadeIn(false), []);

    // manually adding the styles to fade back in
    if (!fadeIn) setTimeout(() => setFadeIn(true), 50);
    const style = fadeIn
        ? {
              opacity: 1,
              transition: '.15s opacity ease-in',
              transitionDelay: '.25s'
          }
        : {
              opacity: 0
          };

    return (
        <div className="flex flex-1 flex-col bg-base-200" style={style}>
            <PageHeader header="Temp Header" subheader="temp subheader" />
            <EntityTabs entityType={pageEntityType} listType={pageListType} />
            <div className="flex flex-1 w-full h-full bg-base-100 relative z-0 overflow-hidden">
                <div
                    className={`${sidePanelOpen ? 'overlay' : ''} h-full w-full overflow-auto`}
                    id="capture-list"
                >
                    <EntityComponent
                        entityType={pageEntityType}
                        entityId={pageEntityId}
                        entityListType={pageListType}
                        search={pageSearch}
                    />
                </div>
                <searchContext.Provider value={searchContexts.sidePanel}>
                    <SidePanelAnimation condition={sidePanelOpen}>
                        <WorkflowSidePanel stateStack={sidePanelStateStack}>
                            {sidePanelOpen ? (
                                <EntityComponent
                                    entityId={sidePanelEntityId}
                                    entityType={sidePanelEntityType}
                                    listType={sidePanelListType}
                                    search={sidePanelSearch}
                                    entityContext={{
                                        [pageEntity.entityType]: pageEntity.entityId
                                    }}
                                />
                            ) : (
                                <span />
                            )}
                        </WorkflowSidePanel>
                    </SidePanelAnimation>
                </searchContext.Provider>
            </div>
        </div>
    );
};

export default withRouter(WorkflowEntityPageLayout);
