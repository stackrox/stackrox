import pluralize from 'pluralize';

import entityLabels from 'messages/entity';

const getLabel = (entityType) => pluralize(entityLabels[entityType]);

type MenuLinkOption = {
    label: string;
    link: string;
};

type WorkflowState = {
    resetPage: (type: string) => { toUrl: () => string };
};

// creates options for menu links
export function createOptions(
    availableEntityTypes: string[],
    workflowState: WorkflowState
): MenuLinkOption[] {
    return availableEntityTypes.map((entityType) => getOption(entityType, workflowState));
}

export function getOption(type: string, workflowState: WorkflowState): MenuLinkOption {
    return {
        label: getLabel(type),
        link: workflowState.resetPage(type).toUrl(),
    };
}
