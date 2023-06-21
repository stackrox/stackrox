import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import entityLabels from 'messages/entity';
import { useTheme } from 'Containers/ThemeProvider';
import workflowStateContext from 'Containers/workflowStateContext';
import { getVulnerabilityManagementEntityTypesByRelationship } from 'utils/entityRelationships';
import entityTypes from 'constants/entityTypes';
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

    const matches = getVulnerabilityManagementEntityTypesByRelationship(entityType, 'MATCHES')
        .map((matchEntity) => {
            let countKeyToUse = countKeyMap[matchEntity];
            if (countKeyMap[matchEntity].includes('k8sVulnCount')) {
                countKeyToUse = 'vulnCount';
            }
            const count = data[countKeyToUse];

            return {
                count,
                label: pluralize(matchEntity, count).replace('_', ' '),
                entity: matchEntity,
                url: workflowState.pushList(matchEntity).setSearch('').toUrl(),
            };
        })
        .filter((matchObj) => {
            return (
                entityType === entityTypes.CLUSTER_CVE ||
                (matchObj.count && !entityContext[matchObj.entity])
            );
        });
    const contains = getVulnerabilityManagementEntityTypesByRelationship(entityType, 'CONTAINS')
        .map((containEntity) => {
            let countKeyToUse = countKeyMap[containEntity];
            if (countKeyMap[containEntity].includes('k8sVulnCount')) {
                countKeyToUse = 'vulnCount';
            }
            const count = data[countKeyToUse];

            const entityLabel = entityLabels[containEntity].toUpperCase();
            return {
                count,
                label: pluralize(entityLabel, count),
                entity: containEntity,
                url: workflowState.pushList(containEntity).setSearch('').toUrl(),
            };
        })
        .filter((containObj) => {
            return containObj.count && !entityContext[containObj.entity];
        });
    if (!matches.length && !contains.length) {
        return null;
    }
    return (
        <div
            className={`h-full relative border-base-100 border-l max-w-43 ${
                !isDarkMode ? 'bg-primary-300' : 'bg-base-100'
            }`}
        >
            {/* TODO: decide if this should be added as custom tailwind class, or a "component" CSS class in app.tw.css */}
            <div className="sticky top-0 py-4">
                <h2 className="mb-3 p-2 rounded-l text-lg text-base-600 text-center font-700">
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
