import React, { useContext } from 'react';
import PropTypes from 'prop-types';

import workflowStateContext from 'Containers/workflowStateContext';
import { getEntityTypesByRelationship } from 'modules/entityRelationships';
import relationshipTypes from 'constants/relationshipTypes';
import entityTypes from 'constants/entityTypes';
import TileList from 'Components/TileList';
import pluralize from 'pluralize';

const defaultCountKeyMap = {
    [entityTypes.COMPONENT]: 'componentCount',
    [entityTypes.CVE]: 'vulnCount',
    [entityTypes.IMAGE]: 'imageCount',
    [entityTypes.POLICY]: 'policyCount',
    [entityTypes.DEPLOYMENT]: 'deploymentCount',
    [entityTypes.NAMESPACE]: 'namespaceCount'
};

const RelatedEntitiesSideList = ({ entityType, data, altCountKeyMap, entityContext }) => {
    const workflowState = useContext(workflowStateContext);
    const { useCase } = workflowState;
    if (!useCase) return null;

    const countKeyMap = { ...defaultCountKeyMap, ...altCountKeyMap };

    const matches = getEntityTypesByRelationship(entityType, relationshipTypes.MATCHES, useCase)
        .map(matchEntity => {
            const count = data[countKeyMap[matchEntity]];
            return {
                count,
                label: pluralize(matchEntity, count),
                entity: matchEntity,
                url: workflowState
                    .pushList(matchEntity)
                    .setSearch('')
                    .toUrl()
            };
        })
        .filter(matchObj => matchObj.count && !entityContext[matchObj.entity]);
    const contains = getEntityTypesByRelationship(entityType, relationshipTypes.CONTAINS, useCase)
        .map(containEntity => {
            const count = data[countKeyMap[containEntity]];
            return {
                count,
                label: pluralize(containEntity, count),
                entity: containEntity,
                url: workflowState
                    .pushList(containEntity)
                    .setSearch('')
                    .toUrl()
            };
        })
        .filter(containObj => containObj.count && !entityContext[containObj.entity]);
    if (!matches.length && !contains.length) return null;
    return (
        <div className="bg-primary-300 h-full relative border-base-100 border-l w-32">
            {/* TODO: decide if this should be added as custom tailwind class, or a "component" CSS class in app.css */}
            <h2
                style={{
                    position: 'relative',
                    left: '-0.5rem',
                    width: 'calc(100% + 0.5rem)'
                }}
                className="my-4 p-2 bg-primary-700 text-base text-base-100 rounded-l text-lg"
            >
                Related entities
            </h2>
            {!!matches.length && <TileList items={matches} title="Matches" />}
            {!!contains.length && <TileList items={contains} title="Contains" />}
        </div>
    );
};

RelatedEntitiesSideList.propTypes = {
    entityType: PropTypes.string.isRequired,
    data: PropTypes.shape({}).isRequired,
    altCountKeyMap: PropTypes.shape({}),
    entityContext: PropTypes.shape({}).isRequired
};

RelatedEntitiesSideList.defaultProps = {
    altCountKeyMap: {}
};

export default RelatedEntitiesSideList;
