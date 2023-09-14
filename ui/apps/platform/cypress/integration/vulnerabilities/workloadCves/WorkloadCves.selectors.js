const watchedImageLabelText = 'Watched image';

export const selectors = {
    resourceDropdown: '.pf-c-toolbar button[aria-label="resource filter menu toggle"]',
    resourceMenuItem: (resource) =>
        `.pf-c-toolbar ul[aria-label="resource filter menu items"] button:contains("${resource}")`,
    resourceValueTypeahead: (resource) =>
        `.pf-c-toolbar input[aria-label="Filter by ${resource.toUpperCase()}"]`,
    resourceValueMenuItem: (resource, value) =>
        `.pf-c-toolbar ul[aria-label="Filter by ${resource.toUpperCase()}"] button:contains("${value}")`,
    severityDropdown: '.pf-c-toolbar button[aria-label="CVE severity filter menu toggle"]',
    severityMenuItems: '.pf-c-toolbar ul[aria-label="CVE severity filter menu items"]',
    severityMenuItem: (severity) => `${selectors.severityMenuItems} label:contains("${severity}")`,
    fixabilityDropdown: '.pf-c-toolbar button[aria-label="CVE status filter menu toggle"]',
    fixabilityMenuItems: '.pf-c-toolbar ul[aria-label="CVE status filter menu items"]',
    fixabilityMenuItem: (fixability) =>
        `${selectors.fixabilityMenuItems} label:contains("${fixability}")`,
    filterChipGroup: (category) => `.pf-c-toolbar .pf-c-chip-group *:contains("${category}")`,
    filterChipGroupRemove: (category) =>
        `${selectors.filterChipGroup(category)} button[aria-label="close"]`,
    filterChipGroupItem: (category, item) =>
        `${selectors.filterChipGroup(category)} + ul li:contains("${item}")`,
    filterChipGroupItemRemove: (category, item) =>
        `${selectors.filterChipGroupItem(category, item)} button[aria-label="close"]`,
    clearFiltersButton: '.pf-c-toolbar button:contains("Clear filters")',
    filteredViewLabel: '.pf-c-label:contains("Filtered view")',
    entityTypeToggleItem: (entityType) =>
        `.pf-c-toggle-group[aria-label="Entity type toggle items"] button:contains("${entityType}")`,
    summaryCard: (cardTitle) => `.pf-c-card:contains("${cardTitle}")`,
    iconText: (textContent) => `svg ~ *:contains("${textContent}")`,
    firstTableRow: 'table tbody:nth-of-type(1) tr:nth-of-type(1)',
    nonZeroCveSeverityCounts: '*[aria-label*="severity cves"i]:not([aria-label^="0"])',

    // Watched image selectors
    watchedImageLabel: `.pf-c-label:contains("${watchedImageLabelText}")`,
    firstUnwatchedImageRow: `tbody tr:has(td[data-label="Image"]:not(:contains("${watchedImageLabelText}"))):eq(0)`,
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
