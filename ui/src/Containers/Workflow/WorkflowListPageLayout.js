import React, { useContext } from 'react';
import pluralize from 'pluralize';
import startCase from 'lodash/startCase';
import { useQuery } from 'react-apollo';

import PageHeader from 'Components/PageHeader';
import ExportButton from 'Components/ExportButton';
import SidePanelAnimation from 'Components/animations/SidePanelAnimation';
import useCaseTypes from 'constants/useCaseTypes';
import VulnMgmtList from 'Containers/VulnMgmt/List/VulnMgmtList';
import VulnMgmtEntity from 'Containers/VulnMgmt/Entity/VulnMgmtEntity';
import VulnMgmtListQueries from 'Containers/VulnMgmt/List/VulnMgmtListQueries';
import VulnMgmtEntityQueries from 'Containers/VulnMgmt/Entity/VulnMgmtEntityQueries';
import entityLabels from 'messages/entity';
import URLService from 'modules/URLService';

import WorkflowSidePanel from './WorkflowSidePanel';
import workflowStateContext from '../workflowStateContext';

// TODO: extract these map objects to somewhere common and reusable
const ListMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtList
};

const EntityMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtEntity
};

const ListQueryMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtListQueries
};

const EntityQueryMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtEntityQueries
};

const WorkflowListPageLayout = ({ match, location }) => {
    const params = URLService.getParams(match, location);
    const { useCase, stateStack } = useContext(workflowStateContext);

    const List = ListMap[useCase];
    const Entity = EntityMap[useCase];

    const listQueries = ListQueryMap[useCase];
    const entityQueries = EntityQueryMap[useCase];

    const entityListType = stateStack[0] && stateStack[0].entityType;
    const entityId = stateStack[1] && stateStack[1].entityId;

    const header = pluralize(entityLabels[entityListType]);
    const exportFilename = `${pluralize(startCase(header))} Report`;

    const listQueryToUse = listQueries.getListQuery(entityListType);
    const entityQueryToUse = entityQueries.getEntityQuery();

    const { loading: listLoading, error: listError, data: listData } = useQuery(listQueryToUse);

    return (
        <div className="flex flex-col relative min-h-full">
            <PageHeader header={header} subHeader="Entity List">
                <div className="flex flex-1 justify-end">
                    <div className="flex">
                        <div className="flex items-center">
                            <ExportButton
                                fileName={exportFilename}
                                type={entityListType}
                                page="configManagement"
                                pdfId="capture-list"
                            />
                        </div>
                    </div>
                </div>
            </PageHeader>
            <div className="flex flex-1 h-full bg-base-100 relative z-0" id="capture-list">
                <List
                    wrapperClass={`bg-base-100 ${entityId ? 'overlay' : ''}`}
                    entityListType={entityListType}
                    entityId={entityId}
                    {...params}
                    loading={listLoading}
                    error={listError}
                    data={listData}
                />
            </div>
            <SidePanelAnimation condition={!!entityId}>
                <WorkflowSidePanel
                    query={entityQueryToUse}
                    // eslint-disable-next-line react/jsx-no-bind
                    render={({ loading, error, data }) => (
                        <Entity
                            entityType={entityListType}
                            entityId={entityId}
                            loading={loading}
                            error={error}
                            data={data}
                            {...params}
                        />
                    )}
                />
            </SidePanelAnimation>
        </div>
    );
};

export default WorkflowListPageLayout;
