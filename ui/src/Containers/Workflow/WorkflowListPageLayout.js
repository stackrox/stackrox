import React from 'react';
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
    const { context: useCase, pageEntityListType, entityId1 } = params;
    const List = ListMap[useCase];
    const Entity = EntityMap[useCase];
    const listQueries = ListQueryMap[useCase];
    const entityQueries = EntityQueryMap[useCase];

    const header = pluralize(entityLabels[pageEntityListType]);
    const exportFilename = `${pluralize(startCase(header))} Report`;

    const listQueryToUse = listQueries.getListQuery(pageEntityListType);
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
                                type={pageEntityListType}
                                page="configManagement"
                                pdfId="capture-list"
                            />
                        </div>
                    </div>
                </div>
            </PageHeader>
            <div className="flex flex-1 h-full bg-base-100 relative z-0" id="capture-list">
                <List
                    wrapperClass={`bg-base-100 ${entityId1 ? 'overlay' : ''}`}
                    entityListType={pageEntityListType}
                    entityId={entityId1}
                    {...params}
                    loading={listLoading}
                    error={listError}
                    data={listData}
                />
            </div>
            <SidePanelAnimation condition={!!entityId1}>
                <WorkflowSidePanel
                    query={entityQueryToUse}
                    entityId1={entityId1}
                    entityType1={pageEntityListType}
                    // eslint-disable-next-line react/jsx-no-bind
                    render={({ loading, error, data }) => (
                        <Entity
                            entityType={pageEntityListType}
                            entityId={entityId1}
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
