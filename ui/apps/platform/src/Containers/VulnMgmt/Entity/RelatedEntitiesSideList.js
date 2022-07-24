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
import useFeatureFlags from 'hooks/useFeatureFlags';
import filterEntityRelationship from 'Containers/VulnMgmt/VulnMgmt.utils/filterEntityRelationship';

const RelatedEntitiesSideList = ({ entityType, data, altCountKeyMap, entityContext }) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVMUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');

    const { isDarkMode } = useTheme();
    const workflowState = useContext(workflowStateContext);
    const { useCase } = workflowState;
    if (!useCase) {
        return null;
    }

    const countKeyMap = { ...defaultCountKeyMap, ...altCountKeyMap };

    const matches = getEntityTypesByRelationship(entityType, relationshipTypes.MATCHES, useCase)
        // @TODO: Remove the following filter step once ROX_FRONTEND_VM_UPDATES is ON
        .filter((matchEntity) => {
            return filterEntityRelationship(showVMUpdates, matchEntity);
        })
        .map((matchEntity) => {
            let countKeyToUse = countKeyMap[matchEntity];
            if (
                countKeyMap[matchEntity].includes('imageComponentCount') ||
                countKeyMap[matchEntity].includes('nodeComponentCount')
            ) {
                countKeyToUse = 'componentCount';
            }
            if (
                countKeyMap[matchEntity].includes('imageVulnerabilityCount') ||
                countKeyMap[matchEntity].includes('nodeVulnerabilityCount') ||
                countKeyMap[matchEntity].includes('clusterVulnerabilityCount') ||
                countKeyMap[matchEntity].includes('k8sVulnCount')
            ) {
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
            return matchObj.count && !entityContext[matchObj.entity];
        });
    const contains = getEntityTypesByRelationship(entityType, relationshipTypes.CONTAINS, useCase)
        // @TODO: Remove the following filter step once ROX_FRONTEND_VM_UPDATES is ON
        .filter((containEntity) => {
            return filterEntityRelationship(showVMUpdates, containEntity);
        })
        .map((containEntity) => {
            let countKeyToUse = countKeyMap[containEntity];
            if (
                countKeyMap[containEntity].includes('imageComponentCount') ||
                countKeyMap[containEntity].includes('nodeComponentCount')
            ) {
                countKeyToUse = 'componentCount';
            }
            if (
                countKeyMap[containEntity].includes('imageVulnerabilityCount') ||
                countKeyMap[containEntity].includes('nodeVulnerabilityCount') ||
                countKeyMap[containEntity].includes('clusterVulnerabilityCount') ||
                countKeyMap[containEntity].includes('k8sVulnCount')
            ) {
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
                <h2
                    style={{
                        position: 'relative',
                        left: '-0.5rem',
                        width: 'calc(100% + 0.5rem)',
                    }}
                    className={`mb-3 p-2 rounded-l text-lg ${
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
