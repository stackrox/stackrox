import React, { useEffect, useState } from 'react';
import { withRouter } from 'react-router-dom';
import startCase from 'lodash/startCase';

import SidePanelAnimatedArea from 'Components/animations/SidePanelAnimatedArea';
import PageHeader from 'Components/PageHeader';
import EntitiesMenu from 'Components/workflow/EntitiesMenu';
import ExportButton from 'Components/ExportButton';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import { useTheme } from 'Containers/ThemeProvider';
import workflowStateContext from 'Containers/workflowStateContext';
import entityTypes from 'constants/entityTypes';
import parseURL from 'utils/URLParser';
import getSidePanelEntity from 'utils/getSidePanelEntity';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import { WorkflowState } from 'utils/WorkflowState';
import { getVulnerabilityManagementEntityTypes } from 'utils/entityRelationships';
import useCaseLabels from 'messages/useCase';
import useEntityName from 'hooks/useEntityName';
import { exportCvesAsCsv } from 'services/VulnerabilitiesService';

import { entityNounSentenceCaseSingular } from '../entitiesForVulnerabilityManagement';
import WorkflowSidePanel from '../WorkflowSidePanel';
import EntityComponent from './VulnMgmtEntity';
import EntityTabs from './EntityTabs';

const WorkflowEntityPageLayout = ({ location }) => {
    const [isExporting, setIsExporting] = useState(false);
    const { isDarkMode } = useTheme();

    const workflowState = parseURL(location);
    const { stateStack, useCase, search } = workflowState;
    const pageState = new WorkflowState(useCase, workflowState.getPageStack(), search);

    // set up cache-busting system that either the list or sidepanel can use to trigger list refresh
    const [refreshTrigger, setRefreshTrigger] = useState(0);

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
        return exportCvesAsCsv(fileName, workflowState, pageListType);
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

    const subheaderText = entityNounSentenceCaseSingular[pageEntityType];
    const { entityName = '' } = useEntityName(pageEntityType, pageEntityId);
    const entityContext = {};

    const exportFilename = `${useCaseLabels[useCase]} ${startCase(
        subheaderText
    )}: ${entityName} Report`;

    if (pageEntity) {
        entityContext[pageEntity.entityType] = pageEntity.entityId;
    }

    const pdfId = pageListType ? 'capture-list' : 'capture-widgets';

    return (
        <workflowStateContext.Provider value={pageState}>
            <div className="flex flex-1 flex-col" style={style}>
                <PageHeader
                    header={entityName}
                    subHeader={subheaderText}
                    classes="pr-0 ignore-react-onclickoutside"
                >
                    <div className="flex flex-1 justify-end h-full">
                        {(pageListType === entityTypes.IMAGE_CVE ||
                            pageListType === entityTypes.NODE_CVE ||
                            pageListType === entityTypes.CLUSTER_CVE) && (
                            <div className="flex items-center pr-2">
                                <ExportButton
                                    fileName={exportFilename}
                                    type={pageListType}
                                    page={useCase}
                                    disabled={!!sidePanelEntityId}
                                    customCsvExportHandler={customCsvExportHandler}
                                    isExporting={isExporting}
                                    setIsExporting={setIsExporting}
                                />
                            </div>
                        )}
                        <div className="flex items-center">
                            <EntitiesMenu
                                text="All Entities"
                                options={getVulnerabilityManagementEntityTypes()}
                            />
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
            {isExporting && <BackdropExporting />}
        </workflowStateContext.Provider>
    );
};

export default withRouter(WorkflowEntityPageLayout);
