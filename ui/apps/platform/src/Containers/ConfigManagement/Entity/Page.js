import React, { useState, useEffect } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';

import SidePanelAnimatedArea from 'Components/animations/SidePanelAnimatedArea';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import { searchParams } from 'constants/searchParams';
import configMgmtPaginationContext, {
    MAIN_PAGINATION_PARAMS,
    SIDEPANEL_PAGINATION_PARAMS,
} from 'Containers/configMgmtPaginationContext';
import searchContext from 'Containers/searchContext';
import { useTheme } from 'Containers/ThemeProvider';
import workflowStateContext from 'Containers/workflowStateContext';
import parseURL from 'utils/URLParser';
import URLService from 'utils/URLService';
import { WorkflowState } from 'utils/WorkflowState';
import EntityPageHeader from './EntityPageHeader';
import Tabs from './EntityTabs';
import SidePanel from '../SidePanel/SidePanel';
import Entity from '../Entity';

const EntityPage = ({ match, location }) => {
    const [isExporting, setIsExporting] = useState(false);
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

    const params = URLService.getParams(match, location);
    const { urlParams } = URLService.getURL(match, location);
    const {
        pageEntityType,
        pageEntityId,
        entityListType1,
        entityType1,
        entityId1,
        entityType2,
        entityListType2,
        entityId2,
        query,
    } = params;
    const [fadeIn, setFadeIn] = useState(false);

    useEffect(() => setFadeIn(false), [pageEntityId]);

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
    return (
        <workflowStateContext.Provider value={pageState}>
            <div className="flex flex-1 flex-col" style={style}>
                <EntityPageHeader
                    entityType={pageEntityType}
                    entityId={pageEntityId}
                    urlParams={urlParams}
                    isExporting={isExporting}
                    setIsExporting={setIsExporting}
                />
                <Tabs
                    pageEntityId={pageEntityId}
                    entityType={pageEntityType}
                    entityListType={entityListType1}
                    disabled={!!entityId1}
                />
                <div className="flex flex-1 w-full h-full relative z-0 overflow-hidden">
                    <configMgmtPaginationContext.Provider value={MAIN_PAGINATION_PARAMS}>
                        <div className="h-full w-full overflow-auto" id="capture-list">
                            <Entity
                                entityType={pageEntityType}
                                entityId={pageEntityId}
                                entityListType={entityListType1}
                                entityId1={entityId1}
                                query={query}
                            />
                        </div>
                    </configMgmtPaginationContext.Provider>
                    <searchContext.Provider value={searchParams.sidePanel}>
                        <configMgmtPaginationContext.Provider value={SIDEPANEL_PAGINATION_PARAMS}>
                            <SidePanelAnimatedArea isDarkMode={isDarkMode} isOpen={!!entityId1}>
                                <SidePanel
                                    contextEntityId={pageEntityId}
                                    contextEntityType={pageEntityType}
                                    entityListType1={entityListType1}
                                    entityType1={entityType1}
                                    entityId1={entityId1}
                                    entityType2={entityType2}
                                    entityListType2={entityListType2}
                                    entityId2={entityId2}
                                    query={query}
                                />
                            </SidePanelAnimatedArea>
                        </configMgmtPaginationContext.Provider>
                    </searchContext.Provider>
                </div>
            </div>
            {isExporting && <BackdropExporting />}
        </workflowStateContext.Provider>
    );
};

EntityPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
};

export default withRouter(EntityPage);
