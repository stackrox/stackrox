import React, { useCallback, useContext, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom-v5-compat';
import pluralize from 'pluralize';
import upperFirst from 'lodash/upperFirst';
import startCase from 'lodash/startCase';

import SidePanelAnimatedArea from 'Components/animations/SidePanelAnimatedArea';
import PageHeader from 'Components/PageHeader';
import { PageBody } from 'Components/Panel';
import EntitiesMenu from 'Components/workflow/EntitiesMenu';
import ExportButton from 'Components/ExportButton';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import configMgmtPaginationContext, {
    MAIN_PAGINATION_PARAMS,
    SIDEPANEL_PAGINATION_PARAMS,
} from 'Containers/configMgmtPaginationContext';
import searchContext from 'Containers/searchContext';
import { searchParams } from 'constants/searchParams';
import workflowStateContext from 'Containers/workflowStateContext';
import useClickOutside from 'hooks/useClickOutside';
import useWorkflowMatch from 'hooks/useWorkflowMatch';
import entityLabels from 'messages/entity';
import parseURL from 'utils/URLParser';
import URLService from 'utils/URLService';
import { getConfigurationManagementEntityTypes } from 'utils/entityRelationships';
import { WorkflowState } from 'utils/WorkflowState';
import EntityList from './EntityList';
import SidePanel from '../SidePanel/SidePanel';

const ListPage = () => {
    const sidePanelRef = useRef(null);
    const location = useLocation();
    const navigate = useNavigate();
    const match = useWorkflowMatch();
    const [isExporting, setIsExporting] = useState(false);

    const workflowState = parseURL(location);
    const { useCase, search, sort, paging } = workflowState;
    const pageState = new WorkflowState(
        useCase,
        workflowState.getPageStack(),
        search,
        sort,
        paging
    );

    const params = URLService.getParams(match, location);
    const { pageEntityListType, entityId1, entityType2, entityListType2, entityId2, query } =
        params;
    const searchParam = useContext(searchContext);

    const closeSidePanel = useCallback(() => {
        navigate(URLService.getURL(match, location).clearSidePanelParams().url());
    }, [navigate, match, location]);

    useClickOutside(sidePanelRef, closeSidePanel, !!entityId1);

    function onRowClick(entityId) {
        const urlBuilder = URLService.getURL(match, location).push(entityId);
        navigate(urlBuilder.url());
    }

    const header = upperFirst(pluralize(entityLabels[pageEntityListType]));
    const exportFilename = `${pluralize(startCase(header))} Report`;
    return (
        <workflowStateContext.Provider value={pageState}>
            <PageHeader
                header={header}
                subHeader="Entity list"
                classes="pr-0 ignore-react-onclickoutside"
            >
                <div className="flex flex-1 justify-end h-full">
                    <div className="flex items-center">
                        <ExportButton
                            fileName={exportFilename}
                            type={pageEntityListType}
                            page="configManagement"
                            pdfId="capture-list"
                            isExporting={isExporting}
                            setIsExporting={setIsExporting}
                        />
                    </div>
                    <div className="flex items-center pl-2">
                        <EntitiesMenu
                            text="All Entities"
                            options={getConfigurationManagementEntityTypes()}
                        />
                    </div>
                </div>
            </PageHeader>
            <PageBody>
                <configMgmtPaginationContext.Provider value={MAIN_PAGINATION_PARAMS}>
                    <EntityList
                        entityListType={pageEntityListType}
                        entityId={entityId1}
                        onRowClick={onRowClick}
                        query={query[searchParam]}
                    />
                </configMgmtPaginationContext.Provider>
                <searchContext.Provider value={searchParams.sidePanel}>
                    <configMgmtPaginationContext.Provider value={SIDEPANEL_PAGINATION_PARAMS}>
                        <SidePanelAnimatedArea isOpen={!!entityId1}>
                            <div ref={sidePanelRef}>
                                <SidePanel
                                    entityType1={pageEntityListType}
                                    entityId1={entityId1}
                                    entityType2={entityType2}
                                    entityListType2={entityListType2}
                                    entityId2={entityId2}
                                    query={query}
                                />
                            </div>
                        </SidePanelAnimatedArea>
                    </configMgmtPaginationContext.Provider>
                </searchContext.Provider>
            </PageBody>
            {isExporting && <BackdropExporting />}
        </workflowStateContext.Provider>
    );
};

export default ListPage;
