const watchedImageLabelText = 'Watched image';
const filterChipSection = '.pf-c-toolbar .pf-c-toolbar__group[aria-label="applied search filters"]';

export const selectors = {
    // Toolbar selectors
    resourceDropdown: '.pf-c-toolbar button[aria-label="resource filter menu toggle"]',
    resourceMenuItem: (resource) =>
        `.pf-c-toolbar ul[aria-label="resource filter menu items"] button:contains("${resource}")`,
    resourceValueTypeahead: (resource) =>
        `.pf-c-toolbar input[aria-label="Filter by ${resource.toUpperCase()}"]`,
    resourceValueMenuItem: (resource) =>
        `.pf-c-toolbar ul[aria-label="Filter by ${resource.toUpperCase()}"] button`,
    severityDropdown: '.pf-c-toolbar button[aria-label="CVE severity filter menu toggle"]',
    severityMenuItems: '.pf-c-toolbar ul[aria-label="CVE severity filter menu items"]',
    severityMenuItem: (severity) => `${selectors.severityMenuItems} label:contains("${severity}")`,
    fixabilityDropdown: '.pf-c-toolbar button[aria-label="CVE status filter menu toggle"]',
    fixabilityMenuItems: '.pf-c-toolbar ul[aria-label="CVE status filter menu items"]',
    fixabilityMenuItem: (fixability) =>
        `${selectors.fixabilityMenuItems} label:contains("${fixability}")`,
    filterChipGroup: `${filterChipSection} .pf-c-chip-group`,
    filterChipGroupForCategory: (category) =>
        `${selectors.filterChipGroup} *:contains("${category}")`,
    filterChipGroupRemove: (category) =>
        `${selectors.filterChipGroupForCategory(category)} button[aria-label="close"]`,
    filterChipGroupItem: (category, item) =>
        `${selectors.filterChipGroupForCategory(category)} + ul li:contains("${item}")`,
    filterChipGroupItemRemove: (category, item) =>
        `${selectors.filterChipGroupItem(category, item)} button[aria-label="close"]`,
    clearFiltersButton: `${filterChipSection} button:contains("Clear filters")`,

    // General selectors
    filteredViewLabel: '.pf-c-label:contains("Filtered view")',
    entityTypeToggleItem: (entityType) =>
        `.pf-c-toggle-group[aria-label="Entity type toggle items"] button:contains("${entityType}")`,
    summaryCard: (cardTitle) => `.pf-c-card:contains("${cardTitle}")`,
    iconText: (textContent) => `svg ~ *:contains("${textContent}")`,

    // Data table selectors
    isUpdatingTable: '*[aria-busy="true"] table',
    firstTableRow: 'table tbody:nth-of-type(1) tr:nth-of-type(1)',
    nonZeroCveSeverityCounts: '*[aria-label*="severity cves"i]:not([aria-label^="0"])',
    nonZeroImageSeverityCounts:
        'td[data-label="Images by severity"] *[aria-label$="severity"i]:not([aria-label^="0"])',
    nonZeroCveSeverityCount: (severity) =>
        `span[aria-label*="${severity.toLowerCase()} severity CVEs across this"]`,
    nonZeroImageSeverityCount: (severity) =>
        `span[aria-label*="with ${severity.toLowerCase()} severity"]`,
    hiddenSeverityCount: (severity) =>
        `span[aria-label="${severity} severity is hidden by the applied filter"]`,

    // Watched image selectors
    watchedImageLabel: `.pf-c-label:contains("${watchedImageLabelText}")`,
    firstUnwatchedImageRow: `tbody tr:has(td[data-label="Image"]:not(:contains("${watchedImageLabelText}"))):eq(0)`,
    tableRowActionsForImage: (name) =>
        `tbody tr:has(td[data-label="Image"]:contains("${name}")) *[aria-label="Actions"]`,
    watchedImageCellWithName: (name) =>
        `tbody tr td[data-label="Image"]:contains("${name}"):contains("${watchedImageLabelText}")`,
    manageWatchedImagesButton: 'button:contains("Manage watched images")',
    closeWatchedImageDialogButton: '*[role="dialog"] button:contains("Close")',
    addWatchedImageNameInput: '*[role="dialog"] input[id="imageName"]',
    addImageToWatchListButton: 'button:contains("Add image to watch list")',
    currentWatchedImagesTable: '*[role="dialog"] table',
    modalAlertWithText: (text) => `*[role="dialog"] .pf-c-alert:contains("${text}")`,
    currentWatchedImageRow: (name) =>
        `${selectors.currentWatchedImagesTable} tr:has(td:contains("${name}"))`,
    removeImageFromTableButton: (name) =>
        `${selectors.currentWatchedImagesTable} tr:has(td:contains("${name}")) button:contains("Remove watch")`,
};
