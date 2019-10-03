import React from 'react';
import { useQuery } from 'react-apollo';

import PageHeader from 'Components/PageHeader';
import useCaseTypes from 'constants/useCaseTypes';
import VulnMgmtEntity from 'Containers/VulnMgmt/Entity/VulnMgmtEntity';
import VulnMgmtEntityQueries from 'Containers/VulnMgmt/Entity/VulnMgmtEntityQueries';
import VulnMgmtList from 'Containers/VulnMgmt/List/VulnMgmtList';
import URLService from 'modules/URLService';

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
    const params = URLService.getParams(match, location);
    const { context: useCase, pageEntityType, pageEntityId } = params;

    const Entity = EntityMap[useCase];
    const entityQueries = EntityQueryMap[useCase];

    const entityQueryToUse = entityQueries.getQuery();
    const { loading, error, data } = useQuery(entityQueryToUse, {
        variables: { id: pageEntityId }
    });

    const entityName =
        data && data[pageEntityType.toLowerCase()] && data[pageEntityType.toLowerCase()].name;

    return (
        <div className="flex flex-1 flex-col bg-base-200">
            {!!pageEntityType && <PageHeader header={`${entityName}`} subHeader={pageEntityType} />}

            {/* TODO add Tabs component
            <Tabs
                pageEntityId={pageEntityId}
                entityType={pageEntityType}
                entityListType={entityListType1}
                disabled={!!overlay}
            /> */}

            <Entity
                entityType={pageEntityType}
                entityId={pageEntityId}
                loading={loading}
                error={error}
                data={data}
                {...params}
            />
        </div>
    );
};

export default WorkflowEntityPageLayout;
