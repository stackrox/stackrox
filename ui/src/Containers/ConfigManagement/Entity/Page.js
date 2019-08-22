import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import SidePanelAnimation from 'Components/animations/SidePanelAnimation';

import pluralize from 'pluralize';
import ExportButton from 'Components/ExportButton';
import searchContext from 'Containers/searchContext';
import searchContexts from 'constants/searchContexts';
import PageHeader from './EntityPageHeader';
import Tabs from './EntityTabs';
import SidePanel from '../SidePanel/SidePanel';
import Entity from '../Entity';

const EntityPage = ({ match, location }) => {
    const params = URLService.getParams(match, location);
    const {
        pageEntityType,
        pageEntityId,
        entityListType1,
        entityType1,
        entityId1,
        entityType2,
        entityListType2,
        entityId2,
        query
    } = params;

    const exportFilename = `${pluralize(pageEntityType)}`;
    const { urlParams } = URLService.getURL(match, location);
    let pdfId = 'capture-dashboard-stretch';
    if (urlParams.entityListType1) {
        pdfId = 'capture-list';
    }
    const overlay = !!entityId1;
    return (
        <div className="flex flex-1 flex-col bg-base-200">
            <PageHeader entityType={pageEntityType} entityId={pageEntityId}>
                <div className="flex flex-1 justify-end">
                    <div className="flex">
                        <div className="flex items-center">
                            <ExportButton
                                fileName={exportFilename}
                                type={pageEntityType}
                                page="configManagement"
                                pdfId={pdfId}
                            />
                        </div>
                    </div>
                </div>
            </PageHeader>
            <Tabs
                pageEntityId={pageEntityId}
                entityType={pageEntityType}
                entityListType={entityListType1}
                disabled={!!overlay}
            />
            <div className="flex flex-1 w-full h-full bg-base-100 relative z-0 overflow-auto">
                <div className={`${overlay ? 'overlay' : ''} h-full w-full`}>
                    <Entity
                        entityType={pageEntityType}
                        entityId={pageEntityId}
                        entityListType={entityListType1}
                        query={query}
                    />
                </div>
                <searchContext.Provider value={searchContexts.sidePanel}>
                    <SidePanelAnimation condition={!!entityId1}>
                        <SidePanel
                            className="w-full h-full bg-base-100 border-l border-base-400 shadow-sidepanel"
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
                    </SidePanelAnimation>
                </searchContext.Provider>
            </div>
        </div>
    );
};

EntityPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

export default withRouter(EntityPage);
