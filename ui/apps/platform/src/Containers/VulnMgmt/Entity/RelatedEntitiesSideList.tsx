import React, { ReactNode, useContext } from 'react';

import { defaultCountKeyMap as countKeyMap } from 'constants/workflowPages.constants';
import { useTheme } from 'Containers/ThemeProvider';
import workflowStateContext from 'Containers/workflowStateContext';
import {
    VulnerabilityManagementEntityType,
    getVulnerabilityManagementEntityTypesByRelationship,
} from 'utils/entityRelationships';

import TileList from './TileList';

export type RelatedEntitiesSideListProps = {
    entityType: VulnerabilityManagementEntityType;
    data: Record<string, number>; // enough truth, although not the whole truth
    entityContext: Record<string, string>; // ditto
};

function RelatedEntitiesSideList({
    entityType,
    data,
    entityContext,
}: RelatedEntitiesSideListProps): ReactNode {
    const { isDarkMode } = useTheme();
    const workflowState = useContext(workflowStateContext);
    const { useCase } = workflowState;
    if (!useCase) {
        return null;
    }

    const matches = getVulnerabilityManagementEntityTypesByRelationship(entityType, 'MATCHES')
        .map((matchEntity) => {
            const count = data[countKeyMap[matchEntity]];

            return {
                count,
                entityType: matchEntity,
                url: workflowState.pushList(matchEntity).setSearch('').toUrl(),
            };
        })
        .filter((matchObj) => {
            return (
                entityType === 'CLUSTER_CVE' ||
                (matchObj.count && !entityContext[matchObj.entityType])
            );
        });
    const contains = getVulnerabilityManagementEntityTypesByRelationship(entityType, 'CONTAINS')
        .map((containEntity) => {
            const count = data[countKeyMap[containEntity]];

            return {
                count,
                entityType: containEntity,
                url: workflowState.pushList(containEntity).setSearch('').toUrl(),
            };
        })
        .filter((containObj) => {
            return containObj.count && !entityContext[containObj.entityType];
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
            <div className="sticky top-0 py-4">
                <h2 className="mb-3 p-2 rounded-l text-lg text-base-600 text-center font-700">
                    Related entities
                </h2>
                {!!matches.length && <TileList items={matches} title="Matches" />}
                {!!contains.length && <TileList items={contains} title="Contains" />}
            </div>
        </div>
    );
}

export default RelatedEntitiesSideList;
