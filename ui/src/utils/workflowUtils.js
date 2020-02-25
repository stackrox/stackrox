import pluralize from 'pluralize';

import entityLabels from 'messages/entity';

const getLabel = entityType => pluralize(entityLabels[entityType]);

// creates options for menu links
export function createOptions(entityTypes, workflowState) {
    return entityTypes.map(entityType => getOption(entityType, workflowState));
}

export function getOption(type, workflowState) {
    return {
        label: getLabel(type),
        link: workflowState.resetPage(type).toUrl()
    };
}
