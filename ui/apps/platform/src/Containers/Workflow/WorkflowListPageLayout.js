import React, { useState } from 'react';
import pluralize from 'pluralize';
import startCase from 'lodash/startCase';

import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import PageHeader from 'Components/PageHeader';
import { useTheme } from 'Containers/ThemeProvider';
import SidePanelAnimatedArea from 'Components/animations/SidePanelAnimatedArea';
import ExportButton from 'Components/ExportButton';
import EntitiesMenu from 'Components/workflow/EntitiesMenu';
import entityTypes from 'constants/entityTypes';
import useFeatureFlags from 'hooks/useFeatureFlags';
import entityLabels from 'messages/entity';
import useCaseLabels from 'messages/useCase';
import getSidePanelEntity from 'utils/getSidePanelEntity';
import parseURL from 'utils/URLParser';
import workflowStateContext from 'Containers/workflowStateContext';
import { WorkflowState } from 'utils/WorkflowState';
import { getUseCaseEntityMap } from 'utils/entityRelationships';
import { exportCvesAsCsv } from 'services/VulnerabilitiesService';
import WorkflowSidePanel from './WorkflowSidePanel';
import { EntityComponentMap, ListComponentMap } from './UseCaseComponentMaps';

const WorkflowListPageLayout = ({ location }) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const useCaseEntityMap = getUseCaseEntityMap();
    if (isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES')) {
        const newTypes = useCaseEntityMap['vulnerability-management'].filter(
            (entityType) => entityType !== entityTypes.COMPONENT
        );
        useCaseEntityMap['vulnerability-management'] = newTypes;
    }

    const { isDarkMode } = useTheme();
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

    function customCsvExportHandler(fileName) {
        return exportCvesAsCsv(fileName, workflowState);
    }

    // Get the list / entity components
    const ListComponent = ListComponentMap[useCase];
    const EntityComponent = EntityComponentMap[useCase];

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

    const header = pluralize(entityLabels[pageListType]);
    const exportFilename = `${useCaseLabels[useCase]} ${pluralize(startCase(header))} Report`;
    const entityContext = {};

    if (selectedRow) {
        const { entityType, entityId } = selectedRow;
        entityContext[entityType] = entityId;
    }

    // TODO: remove all this feature flag check after VM updates have been live for one release
    const showVmUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');
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
            <div className="flex flex-col relative h-full">
                <PageHeader
                    header={header}
                    subHeader="Entity List"
                    classes="pr-0 ignore-react-onclickoutside"
                >
                    <div className="flex flex-1 justify-end h-full">
                        <div className="flex items-center pr-2">
                            <ExportButton
                                fileName={exportFilename}
                                type={pageListType}
                                page={useCase}
                                disabled={!!sidePanelEntityId}
                                pdfId="capture-list"
                                customCsvExportHandler={customCsvExportHandler}
                            />
                        </div>
                        <div className="flex items-center pl-2">
                            <EntitiesMenu text="All Entities" options={useCaseOptions} />
                        </div>
                    </div>
                </PageHeader>
                <div
                    className={`h-full relative z-0 min-h-0 ${
                        !isDarkMode ? 'bg-base-100' : 'bg-base-0'
                    }`}
                    id="capture-list"
                >
                    <ListComponent
                        entityListType={pageListType}
                        selectedRowId={selectedRow && selectedRow.entityId}
                        search={pageSearch}
                        sort={pageSort}
                        page={pagePaging}
                        refreshTrigger={refreshTrigger}
                        setRefreshTrigger={setRefreshTrigger}
                    />
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

export default WorkflowListPageLayout;
