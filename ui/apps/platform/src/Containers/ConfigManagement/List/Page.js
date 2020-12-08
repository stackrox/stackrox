import React, { useContext } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import pluralize from 'pluralize';
import startCase from 'lodash/startCase';

import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import SidePanelAnimatedDiv from 'Components/animations/SidePanelAnimatedDiv';
import PageHeader from 'Components/PageHeader';
import EntitiesMenu from 'Components/workflow/EntitiesMenu';
import ExportButton from 'Components/ExportButton';
import configMgmtPaginationContext, {
    MAIN_PAGINATION_PARAMS,
    SIDEPANEL_PAGINATION_PARAMS,
} from 'Containers/configMgmtPaginationContext';
import searchContext from 'Containers/searchContext';
import { searchParams } from 'constants/searchParams';
import workflowStateContext from 'Containers/workflowStateContext';
import entityLabels from 'messages/entity';
import parseURL from 'utils/URLParser';
import URLService from 'utils/URLService';
import { getUseCaseEntityMap } from 'utils/entityRelationships';
import { WorkflowState } from 'utils/WorkflowState';
import EntityList from './EntityList';
import SidePanel from '../SidePanel/SidePanel';

const ListPage = ({ match, location, history }) => {
    const hostScanningEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_HOST_SCANNING);
    const featureFlags = {
        [knownBackendFlags.ROX_HOST_SCANNING]: hostScanningEnabled,
    };
    const useCaseEntityMap = getUseCaseEntityMap(featureFlags);

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
    const {
        pageEntityListType,
        entityId1,
        entityType2,
        entityListType2,
        entityId2,
        query,
    } = params;
    const searchParam = useContext(searchContext);

    function onRowClick(entityId) {
        const urlBuilder = URLService.getURL(match, location).push(entityId);
        history.push(urlBuilder.url());
    }

    const header = pluralize(entityLabels[pageEntityListType]);
    const exportFilename = `${pluralize(startCase(header))} Report`;
    return (
        <workflowStateContext.Provider value={pageState}>
            <PageHeader
                header={header}
                subHeader="Entity List"
                classes="pr-0 ignore-react-onclickoutside"
            >
                <div className="flex flex-1 justify-end h-full">
                    <div className="flex items-center">
                        <ExportButton
                            fileName={exportFilename}
                            type={pageEntityListType}
                            page="configManagement"
                            pdfId="capture-list"
                        />
                    </div>
                    <div className="flex items-center pl-2">
                        <EntitiesMenu text="All Entities" options={useCaseEntityMap[useCase]} />
                    </div>
                </div>
            </PageHeader>
            <div className="flex flex-1 h-full relative z-0">
                <configMgmtPaginationContext.Provider value={MAIN_PAGINATION_PARAMS}>
                    <EntityList
                        className={entityId1 ? 'overlay' : ''}
                        entityListType={pageEntityListType}
                        entityId={entityId1}
                        onRowClick={onRowClick}
                        query={query[searchParam]}
                    />
                </configMgmtPaginationContext.Provider>
                <searchContext.Provider value={searchParams.sidePanel}>
                    <configMgmtPaginationContext.Provider value={SIDEPANEL_PAGINATION_PARAMS}>
                        <SidePanelAnimatedDiv isOpen={!!entityId1}>
                            <SidePanel
                                className="w-full h-full bg-base-100 border-l border-base-400 shadow-sidepanel"
                                entityType1={pageEntityListType}
                                entityId1={entityId1}
                                entityType2={entityType2}
                                entityListType2={entityListType2}
                                entityId2={entityId2}
                                query={query}
                            />
                        </SidePanelAnimatedDiv>
                    </configMgmtPaginationContext.Provider>
                </searchContext.Provider>
            </div>
        </workflowStateContext.Provider>
    );
};

ListPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
};

export default ListPage;
