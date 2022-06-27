import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import entityLabels from 'messages/entity';
import { useTheme } from 'Containers/ThemeProvider';
import workflowStateContext from 'Containers/workflowStateContext';
import { getEntityTypesByRelationship } from 'utils/entityRelationships';
import relationshipTypes from 'constants/relationshipTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import TileList from 'Components/TileList';

const RelatedEntitiesSideList = ({ entityType, data, altCountKeyMap, entityContext }) => {
    const { isDarkMode } = useTheme();
    const workflowState = useContext(workflowStateContext);
    const { useCase } = workflowState;
    if (!useCase) {
        return null;
    }

    const countKeyMap = { ...defaultCountKeyMap, ...altCountKeyMap };

    const matches = getEntityTypesByRelationship(entityType, relationshipTypes.MATCHES, useCase)
        .map((matchEntity) => {
            const count = data[countKeyMap[matchEntity]];
            return {
                count,
                label: pluralize(matchEntity, count),
                entity: matchEntity,
                url: workflowState.pushList(matchEntity).setSearch('').toUrl(),
            };
        })
        .filter((matchObj) => matchObj.count && !entityContext[matchObj.entity]);
    const contains = getEntityTypesByRelationship(entityType, relationshipTypes.CONTAINS, useCase)
        .map((containEntity) => {
            const count = data[countKeyMap[containEntity]];
            const entityLabel = entityLabels[containEntity].toUpperCase();
            return {
                count,
                label: pluralize(entityLabel, count),
                entity: containEntity,
                url: workflowState.pushList(containEntity).setSearch('').toUrl(),
            };
        })
        .filter((containObj) => containObj.count && !entityContext[containObj.entity]);
    if (!matches.length && !contains.length) {
        return null;
    }
    return (
        <div
            className={` h-full relative border-base-100 border-l w-43 ${
                !isDarkMode ? 'bg-primary-300' : 'bg-base-100'
            }`}
        >
            {/* TODO: decide if this should be added as custom tailwind class, or a "component" CSS class in app.tw.css */}
            <div className="sticky top-0 py-4">
                <h2
                    style={{
                        position: 'relative',
                        left: '-0.5rem',
                        width: 'calc(100% + 0.5rem)',
                    }}
                    className={`mb-3 p-2  text-base rounded-l text-lg ${
                        !isDarkMode
                            ? 'bg-primary-700 text-base-100'
                            : 'bg-tertiary-300 text-base-900'
                    }`}
                >
                    Related entities
                </h2>
                {!!matches.length && <TileList items={matches} title="Matches" />}
                {!!contains.length && <TileList items={contains} title="Contains" />}
            </div>
        </div>
    );
};

RelatedEntitiesSideList.propTypes = {
    entityType: PropTypes.string.isRequired,
    data: PropTypes.shape({}).isRequired,
    altCountKeyMap: PropTypes.shape({}),
    entityContext: PropTypes.shape({}).isRequired,
};

RelatedEntitiesSideList.defaultProps = {
    altCountKeyMap: {},
};

export default RelatedEntitiesSideList;
