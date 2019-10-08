import React, { useContext } from 'react';
import { useQuery } from 'react-apollo';

import PageHeader from 'Components/PageHeader';
import useCaseTypes from 'constants/useCaseTypes';
import VulnMgmtEntity from 'Containers/VulnMgmt/Entity/VulnMgmtEntity';
import VulnMgmtEntityQueries from 'Containers/VulnMgmt/Entity/VulnMgmtEntityQueries';
import VulnMgmtList from 'Containers/VulnMgmt/List/VulnMgmtList';
import URLService from 'modules/URLService';

import workflowStateContext from '../workflowStateContext';

// TODO: extract these map objects to somewhere common and reusable
const EntityMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtEntity
};

const EntityQueryMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtEntityQueries
};

// TODO: build out lists where needed in the sidebar
// eslint-disable-next-line no-unused-vars
const ListMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtList
};

const WorkflowEntityPageLayout = ({ match, location }) => {
    const workflowState = useContext(workflowStateContext);
    const params = URLService.getParams(match, location);

    const Entity = EntityMap[workflowState.useCase];
    const entityQueries = EntityQueryMap[workflowState.useCase];

    const { entityType, entityId } = workflowState.getCurrentEntity();

    const entityQueryToUse = entityQueries.getEntityQuery();
    const { loading, error, data } = useQuery(entityQueryToUse, {
        variables: { id: entityId }
    });

    const entityName =
        data && data[entityType.toLowerCase()] && data[entityType.toLowerCase()].name;

    return (
        <div className="flex flex-1 flex-col bg-base-200">
            {!!entityType && <PageHeader header={`${entityName}`} subHeader={entityType} />}

            {/* TODO add Tabs component
            <Tabs
                pageEntityId={pageEntityId}
                entityType={pageEntityType}
                entityListType={entityListType1}
                disabled={!!overlay}
            /> */}

            <Entity
                entityType={entityType}
                entityId={entityId}
                loading={loading}
                error={error}
                data={data}
                {...params}
            />
        </div>
    );
};

export default WorkflowEntityPageLayout;
