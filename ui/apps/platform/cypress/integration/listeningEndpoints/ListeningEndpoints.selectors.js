export default {
    deploymentTable: '[aria-label="Deployment results"]',
    tableRowWithValueForColumn: (column, value) =>
        `tr:has(td[data-label="${column}"]:contains("${value}"))`,
    expandableRowToggle: '[aria-label="Details"]',
    processTable: '[aria-label="Listening endpoints for deployment"]',
};
