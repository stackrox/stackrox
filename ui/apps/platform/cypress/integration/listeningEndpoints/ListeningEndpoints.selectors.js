export default {
    entityDropdownToggle: '[aria-label="Search entity selection menu toggle"]',
    entityDropdownMenuItems: '[aria-label="Select an entity to filter by"]',
    filterInputBox: (entity) => `[placeholder="Filter results by ${entity}"]`,
    filterAutocompleteResultItem: '[role="listbox"] button',
    deploymentTable: '[aria-label="Deployment results"]',
    tableRowWithValueForColumn: (column, value) =>
        `tr:has(td[data-label="${column}"]:contains("${value}"))`,
    expandableRowToggle: '[aria-label="Details"]',
    processTable: '[aria-label="Listening endpoints for deployment"]',
};
