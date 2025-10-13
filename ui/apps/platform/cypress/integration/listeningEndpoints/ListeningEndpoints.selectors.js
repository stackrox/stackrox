export default {
    entityDropdownToggle: '[aria-label="Search entity selection menu toggle"]',
    entityDropdownMenuItems: '[aria-label="Select an entity to filter by"]',
    filterInputBox: (entity) => `[aria-label="Search by ${entity}"]`,
    filterAutocompleteResults: (entity) => `[aria-label="Filter by ${entity}"]`,
    deploymentTable: '[aria-label="Deployment results"]',
    tableRowWithValueForColumn: (column, value) =>
        `tr:has(td[data-label="${column}"]:contains("${value}"))`,
    expandableRowToggle: '[aria-label="Details"]',
    processTable: '[aria-label="Listening endpoints for deployment"]',
};
