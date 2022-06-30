import React, { useEffect, useState } from 'react';
import { withRouter } from 'react-router-dom';
import startCase from 'lodash/startCase';

import SidePanelAnimatedArea from 'Components/animations/SidePanelAnimatedArea';
import PageHeader from 'Components/PageHeader';
import EntityTabs from 'Components/workflow/EntityTabs';
import EntitiesMenu from 'Components/workflow/EntitiesMenu';
import ExportButton from 'Components/ExportButton';
import { useTheme } from 'Containers/ThemeProvider';
import workflowStateContext from 'Containers/workflowStateContext';
import parseURL from 'utils/URLParser';
import getSidePanelEntity from 'utils/getSidePanelEntity';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import { WorkflowState } from 'utils/WorkflowState';
import { getUseCaseEntityMap } from 'utils/entityRelationships';
import entityLabels from 'messages/entity';
import useCaseLabels from 'messages/useCase';
import useEntityName from 'hooks/useEntityName';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { exportCvesAsCsv } from 'services/VulnerabilitiesService';
import { shouldUseOriginalCase } from 'utils/workflowUtils';
import entityTypes from 'constants/entityTypes';
import WorkflowSidePanel from './WorkflowSidePanel';
import { EntityComponentMap } from './UseCaseComponentMaps';

const WorkflowEntityPageLayout = ({ location }) => {
    const { isDarkMode } = useTheme();
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVmUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');

    const useCaseEntityMap = getUseCaseEntityMap();
    if (showVmUpdates) {
        const newTypes = useCaseEntityMap['vulnerability-management'].filter(
            (entityType) => entityType !== entityTypes.COMPONENT
        );
        newTypes.push(entityTypes.NODE_COMPONENT);
        newTypes.push(entityTypes.IMAGE_COMPONENT);
        useCaseEntityMap['vulnerability-management'] = newTypes;
    }

    const workflowState = parseURL(location);
    const { stateStack, useCase, search } = workflowState;
    const pageState = new WorkflowState(useCase, workflowState.getPageStack(), search);

    // set up cache-busting system that either the list or sidepanel can use to trigger list refresh
    const [refreshTrigger, setRefreshTrigger] = useState(0);

    // Entity Component
    const EntityComponent = EntityComponentMap[useCase];

    // Page props
    const pageEntity = workflowState.getBaseEntity();
    const { entityId: pageEntityId, entityType: pageEntityType } = pageEntity;
    const pageListType = stateStack[1] && !stateStack[1].entityId && stateStack[1].entityType;
    const pageSearch = workflowState.search[searchParams.page];
    const pageSort = workflowState.sort[sortParams.page];
    const pagePaging = workflowState.paging[pagingParams.page];

    // Sidepanel props
    const { sidePanelEntityId, sidePanelEntityType, sidePanelListType } =
        getSidePanelEntity(workflowState);
    const sidePanelSearch = workflowState.search[searchParams.sidePanel];
    const sidePanelSort = workflowState.sort[sortParams.sidePanel];
    const sidePanelPaging = workflowState.paging[pagingParams.sidePanel];

    function customCsvExportHandler(fileName) {
        return exportCvesAsCsv(fileName, workflowState);
    }

    const [fadeIn, setFadeIn] = useState(false);
    useEffect(() => setFadeIn(false), []);

    // manually adding the styles to fade back in
    if (!fadeIn) {
        setTimeout(() => setFadeIn(true), 50);
    }
    const style = fadeIn
        ? {
              opacity: 1,
              transition: '.15s opacity ease-in',
              transitionDelay: '.25s',
          }
        : {
              opacity: 0,
          };

    const subheaderText = entityLabels[pageEntityType];
    const { entityName = '' } = useEntityName(pageEntityType, pageEntityId);
    const entityContext = {};
    // const useLowercase = pageEntityType === entityTypes.IMAGE;
    const useOriginalCase = shouldUseOriginalCase(entityName, pageEntityType);

    const exportFilename = `${useCaseLabels[useCase]} ${startCase(
        subheaderText
    )}: ${entityName} Report`;

    if (pageEntity) {
        entityContext[pageEntity.entityType] = pageEntity.entityId;
    }

    const pdfId = pageListType ? 'capture-list' : 'capture-widgets';

    // TODO: remove all this feature flag check after VM updates have been live for one release
    const useCaseOptions = useCaseEntityMap[useCase].filter((option) => {
        if (showVmUpdates) {
            if (option === entityTypes.CVE) {
                return false;
            }
        } else if (
            option === entityTypes.IMAGE_CVE ||
            option === entityTypes.NODE_CVE ||
            option === entityTypes.CLUSTER_CVE
        ) {
            return false;
        }
        return true;
    });

    return (
        <workflowStateContext.Provider value={pageState}>
            <div className="flex flex-1 flex-col" style={style}>
                <PageHeader
                    header={entityName}
                    subHeader={subheaderText}
                    classes="pr-0 ignore-react-onclickoutside"
                    lowercaseTitle={useOriginalCase}
                >
                    <div className="flex flex-1 justify-end h-full">
                        <div className="flex items-center pr-2">
                            <ExportButton
                                fileName={exportFilename}
                                type={pageListType}
                                page={useCase}
                                disabled={!!sidePanelEntityId}
                                pdfId={pdfId}
                                customCsvExportHandler={customCsvExportHandler}
                            />
                        </div>
                        <div className="flex items-center">
                            <EntitiesMenu text="All Entities" options={useCaseOptions} />
                        </div>
                    </div>
                </PageHeader>
                <EntityTabs entityType={pageEntityType} activeTab={pageListType} />
                <div className="flex flex-1 w-full h-full relative z-0 overflow-hidden">
                    <div className="h-full w-full overflow-auto" id={pdfId}>
                        <EntityComponent
                            entityType={pageEntityType}
                            entityId={pageEntityId}
                            entityListType={pageListType}
                            search={pageSearch}
                            sort={pageSort}
                            page={pagePaging}
                            entityContext={entityContext}
                            refreshTrigger={refreshTrigger}
                            setRefreshTrigger={setRefreshTrigger}
                        />
                    </div>

                    <SidePanelAnimatedArea isDarkMode={isDarkMode} isOpen={!!sidePanelEntityId}>
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

export default withRouter(WorkflowEntityPageLayout);
