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
        .map((matchEntity) => {
            // @TODO: Modify the actual relationship entities once ROX_FRONTEND_VM_UPDATES is in
            let newMatchEntity = matchEntity;
            if (
                showVMUpdates &&
                (entityType === 'NODE_COMPONENT' || entityType === 'NODE') &&
                matchEntity === 'CVE'
            ) {
                newMatchEntity = 'NODE_CVE';
            } else if (
                showVMUpdates &&
                (entityType === 'IMAGE_COMPONENT' || entityType === 'IMAGE') &&
                matchEntity === 'CVE'
            ) {
                newMatchEntity = 'IMAGE_CVE';
            } else if (showVMUpdates && entityType === 'NODE' && matchEntity === 'COMPONENT') {
                newMatchEntity = 'NODE_COMPONENT';
            } else if (showVMUpdates && entityType === 'IMAGE' && matchEntity === 'COMPONENT') {
                newMatchEntity = 'IMAGE_COMPONENT';
            }

            let countKeyToUse = countKeyMap[newMatchEntity];
            if (
                countKeyMap[newMatchEntity].includes('imageComponentCount') ||
                countKeyMap[newMatchEntity].includes('nodeComponentCount')
            ) {
                countKeyToUse = 'componentCount';
            }
            if (
                countKeyMap[newMatchEntity].includes('imageVulnerabilityCount') ||
                countKeyMap[newMatchEntity].includes('nodeVulnerabilityCount') ||
                countKeyMap[newMatchEntity].includes('clusterVulnerabilityCount') ||
                countKeyMap[newMatchEntity].includes('k8sVulnCount')
            ) {
                countKeyToUse = 'vulnCount';
            }
            const count = data[countKeyToUse];

            return {
                count,
                label: pluralize(newMatchEntity, count).replace('_', ' '),
                entity: newMatchEntity,
                url: workflowState.pushList(newMatchEntity).setSearch('').toUrl(),
            };
        })
        .filter((matchObj) => {
            return matchObj.count && !entityContext[matchObj.entity];
        });
    const contains = getEntityTypesByRelationship(entityType, relationshipTypes.CONTAINS, useCase)
        .map((containEntity) => {
            // @TODO: Modify the actual relationship entities once ROX_FRONTEND_VM_UPDATES is in
            let newContainEntity = containEntity;
            if (
                showVMUpdates &&
                (entityType === 'NODE_COMPONENT' || entityType === 'NODE') &&
                containEntity === 'CVE'
            ) {
                newContainEntity = 'NODE_CVE';
            } else if (
                showVMUpdates &&
                (entityType === 'IMAGE_COMPONENT' || entityType === 'IMAGE') &&
                containEntity === 'CVE'
            ) {
                newContainEntity = 'IMAGE_CVE';
            } else if (showVMUpdates && entityType === 'NODE' && containEntity === 'COMPONENT') {
                newContainEntity = 'NODE_COMPONENT';
            } else if (showVMUpdates && entityType === 'IMAGE' && containEntity === 'COMPONENT') {
                newContainEntity = 'IMAGE_COMPONENT';
            }

            let countKeyToUse = countKeyMap[newContainEntity];
            if (
                countKeyMap[newContainEntity].includes('imageComponentCount') ||
                countKeyMap[newContainEntity].includes('nodeComponentCount')
            ) {
                countKeyToUse = 'componentCount';
            }
            if (
                countKeyMap[newContainEntity].includes('imageVulnerabilityCount') ||
                countKeyMap[newContainEntity].includes('nodeVulnerabilityCount') ||
                countKeyMap[newContainEntity].includes('clusterVulnerabilityCount') ||
                countKeyMap[newContainEntity].includes('k8sVulnCount')
            ) {
                countKeyToUse = 'vulnCount';
            }
            const count = data[countKeyToUse];

            const entityLabel = entityLabels[newContainEntity].toUpperCase();
            return {
                count,
                label: pluralize(entityLabel, count),
                entity: newContainEntity,
                url: workflowState.pushList(newContainEntity).setSearch('').toUrl(),
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
