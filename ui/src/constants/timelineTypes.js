// these show the types of objects we'll display in the timeline graph
export const graphObjectTypes = {
    EVENT: 'EVENT'
};

// these show the types of root Entities in the timeline view
// @TODO: Use this to keep track of what level the timeline view is in
export const rootTypes = {
    DEPLOYMENT: 'DEPLOYMENT',
    POD: 'POD'
};

// these show the types of the list of Entities we'll show in the left-hand section of the timeline view
export const graphTypes = {
    POD: 'POD',
    CONTAINER: 'CONTAINER'
};

export const eventTypes = {
    ALL: 'ALL',
    POLICY_VIOLATION: 'POLICY_VIOLATION',
    PROCESS_ACTIVITY: 'PROCESS_ACTIVITY',
    RESTART: 'RESTART',
    FAILURE: 'FAILURE'
};
