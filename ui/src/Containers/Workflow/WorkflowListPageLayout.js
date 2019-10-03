import React from 'react';
import pluralize from 'pluralize';

import PageHeader from 'Components/PageHeader';
import SidePanelAnimation from 'Components/animations/SidePanelAnimation';
import useCaseTypes from 'constants/useCaseTypes';
import VulnMgmtList from 'Containers/VulnMgmt/List/VulnMgmtList';
import VulnMgmtEntity from 'Containers/VulnMgmt/Entity/VulnMgmtEntity';
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

const EntityQueryMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtEntityQueries
};

const WorkflowListPageLayout = ({ match, location }) => {
    const params = URLService.getParams(match, location);
    const { context: useCase, pageEntityListType, entityId1 } = params;
    const List = ListMap[useCase];
    const Entity = EntityMap[useCase];
    const entityQueries = EntityQueryMap[useCase];

    const header = pluralize(entityLabels[pageEntityListType]);

    const entityQueryToUse = entityQueries.getQuery();

    return (
        <div className="flex flex-col relative min-h-full">
            <PageHeader header={header} subHeader="Entity List">
                <div className="flex flex-1 justify-end">
                    <div className="flex">
                        <div className="flex items-center">Tag and Export buttons go here</div>
                    </div>
                </div>
            </PageHeader>
            <div className="flex flex-1 h-full bg-base-100 relative z-0">
                <List
                    wrapperClass={`bg-base-100 ${entityId1 ? 'overlay' : ''}`}
                    entityListType={pageEntityListType}
                    entityId={entityId1}
                    {...params}
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
