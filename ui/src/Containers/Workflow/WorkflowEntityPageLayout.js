import React, { useEffect, useState } from 'react';
import { withRouter } from 'react-router-dom';
import PageHeader from 'Components/PageHeader';
import EntityTabs from 'Components/EntityTabs';
import workflowStateContext from 'Containers/workflowStateContext';
import { parseURL } from 'modules/URLReadWrite';
import getSidePanelEntity from 'utils/getSidePanelEntity';
import searchContexts from 'constants/searchContexts';
import { WorkflowState } from 'modules/WorkflowStateManager';
import entityLabels from 'messages/entity';
import useEntityName from 'hooks/useEntityName';
import WorkflowSidePanel from './WorkflowSidePanel';
import { EntityComponentMap } from './UseCaseComponentMaps';

const WorkflowEntityPageLayout = ({ location }) => {
    const workflowState = parseURL(location);
    const { stateStack, useCase, search } = workflowState;
    const pageState = new WorkflowState(useCase, workflowState.getPageStack(), search);
    const pageSearch = search[searchContexts.page];
    const EntityComponent = EntityComponentMap[useCase];

    // Calculate page entity props
    const pageEntity = stateStack[0];
    const { entityId: pageEntityId, entityType: pageEntityType } = pageEntity;
    const pageListType = stateStack[1] && stateStack[1].entityType;

    const {
        sidePanelEntityId,
        sidePanelEntityType,
        sidePanelListType,
        sidePanelSearch
    } = getSidePanelEntity(workflowState);
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

    const subheaderText = entityLabels[pageEntityType];
    const { entityName = '' } = useEntityName(pageEntityType, pageEntityId);

    return (
        <workflowStateContext.Provider value={pageState}>
            <div className="flex flex-1 flex-col bg-base-200" style={style}>
                <PageHeader header={entityName} subHeader={subheaderText} />
                <EntityTabs entityType={pageEntityType} activeTab={pageListType} />
                <div className="flex flex-1 w-full h-full bg-base-100 relative z-0 overflow-hidden">
                    <div
                        className={`${
                            sidePanelEntityId ? 'overlay' : ''
                        } h-full w-full overflow-auto`}
                        id="capture-list"
                    >
                        <EntityComponent
                            entityType={pageEntityType}
                            entityId={pageEntityId}
                            entityListType={pageListType}
                            search={pageSearch}
                        />
                    </div>

                    <WorkflowSidePanel isOpen={!!sidePanelEntityId}>
                        {sidePanelEntityId ? (
                            <EntityComponent
                                entityId={sidePanelEntityId}
                                entityType={sidePanelEntityType}
                                entityListType={sidePanelListType}
                                search={sidePanelSearch}
                                entityContext={{
                                    [pageEntity.entityType]: pageEntity.entityId
                                }}
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

export default withRouter(WorkflowEntityPageLayout);
