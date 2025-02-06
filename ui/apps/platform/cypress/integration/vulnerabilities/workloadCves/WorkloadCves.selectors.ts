const watchedImageLabelText = 'Watched image';
const filterChipSection =
    '.pf-v5-c-toolbar .pf-v5-c-toolbar__group[aria-label="applied search filters"]';

export const selectors = {
    // Toolbar selectors
    searchOptionsDropdown:
        '.pf-v5-c-toolbar button[aria-label="search options filter menu toggle"]',
    searchOptionsMenuItem: (searchOption) =>
        `.pf-v5-c-toolbar ul[aria-label="search options filter menu items"] button:contains("${searchOption}")`,
    searchOptionsValueTypeahead: (searchOption) =>
        `.pf-v5-c-toolbar input[aria-label="Filter by ${searchOption}"]`,
    searchOptionsValueMenuItem: (searchOption) =>
        `.pf-v5-c-toolbar ul[aria-label="Filter by ${searchOption}"] button`,
    severityDropdown: '.pf-v5-c-toolbar button[aria-label="CVE severity filter menu toggle"]',
    severityMenuItems: '.pf-v5-c-toolbar ul[aria-label="CVE severity filter menu items"]',
    severityMenuItem: (severity) => `${selectors.severityMenuItems} label:contains("${severity}")`,
    fixabilityDropdown: '.pf-v5-c-toolbar button[aria-label="CVE status filter menu toggle"]',
    fixabilityMenuItems: '.pf-v5-c-toolbar ul[aria-label="CVE status filter menu items"]',
    fixabilityMenuItem: (fixability) =>
        `${selectors.fixabilityMenuItems} label:contains("${fixability}")`,
    filterChipGroup: `${filterChipSection} .pf-v5-c-chip-group`,
    filterChipGroupForCategory: (category) =>
        `${selectors.filterChipGroup} *:contains("${category}")`,
    filterChipGroupRemove: (category) =>
        `${selectors.filterChipGroupForCategory(category)} button[aria-label="close"]`,
    filterChipGroupItems: (category) => `${selectors.filterChipGroupForCategory(category)} + ul li`,
    filterChipGroupItem: (category, item) =>
        `${selectors.filterChipGroupForCategory(category)} + ul li:contains("${item}")`,
    filterChipGroupItemRemove: (category, item) =>
        `${selectors.filterChipGroupItem(category, item)} button[aria-label="close"]`,

    searchEntityDropdown:
        '.pf-v5-c-toolbar button[aria-label="compound search filter entity selector toggle"]',
    searchEntityMenuItem:
        '.pf-v5-c-toolbar [aria-label="compound search filter entity selector menu"] button',
    searchAttributeDropdown:
        '.pf-v5-c-toolbar button[aria-label="compound search filter attribute selector toggle"]',
    searchAttributeMenuItem:
        '.pf-v5-c-toolbar [aria-label="compound search filter attribute selector menu"] button',
    searchValueTypeahead: '.pf-v5-c-toolbar input[aria-label^="Filter results by"]',
    searchValueMenuItem: '.pf-v5-c-toolbar [aria-label="Filter results select menu"] button',
    searchValueApplyButton:
        '.pf-v5-c-toolbar button[aria-label="Apply autocomplete input to search"]',

    // General selectors
    filteredViewLabel: '.pf-v5-c-label:contains("Filtered view")',
    iconText: (textContent) => `svg ~ *:contains("${textContent}")`,
    bulkActionMenuToggle: 'button:contains("Bulk actions")',
    menuOption: (optionText) => `*[role="menu"] button:contains("${optionText}")`,
    severityIcon: (severity) => `span.pf-v5-c-icon:contains('${severity}')`,
    loadingSpinner: 'svg[role="progressbar"][aria-valuetext="Loading..."]',

    // Image/Deployment tab selectors
    vulnerabilitiesTab: 'button[role="tab"]:contains("Vulnerabilities")',
    resourcesTab: 'button[role="tab"]:contains("Resources")',

    // Data table selectors
    isUpdatingTable: '*[aria-busy="true"] table',
    tableWithLoadingSpinner: 'table tbody svg[aria-label="Loading table data"]',
    nthTableRow: (n) =>
        `.workload-cves-table-container > table > tbody:nth-of-type(${n}) > tr:nth-of-type(1)`,
    firstTableRow: 'table tbody:nth-of-type(1) tr:nth-of-type(1)',
    tableRowSelectCheckbox: 'td input[type="checkbox"][aria-label^="Select row"]',
    tableRowSelectAllCheckbox: 'thead input[type="checkbox"][aria-label^="Select all rows"]',
    tableRowMenuToggle: 'td button[aria-label="Kebab toggle"]',
    nonZeroImageSeverityCounts:
        'td[data-label="Images by severity"] *[aria-label$="severity"i]:not([aria-label^="0"])',
    nonZeroCveSeverityCount: (severity) =>
        `span[aria-label*="${severity.toLowerCase()} severity cve count across this"]`,
    nonZeroImageSeverityCount: (severity) =>
        `span[aria-label*="with ${severity.toLowerCase()} severity"]`,
    hiddenSeverityCount: `span[aria-label$="severity is hidden by the applied filter"]`,

    // Exception flow selectors
    deferCveModal: '*[role="dialog"]:contains("Request deferral for")',
    markCveFalsePositiveModal: '*[role="dialog"]:contains("Request false positive for")',
    exceptionOptionsTab: 'button[role="tab"]:contains("Options")',
    cveSelectionTab: 'button[role="tab"]:contains("CVE selections")',

    // Watched image selectors
    watchedImageLabel: `.pf-v5-c-label:contains("${watchedImageLabelText}")`,
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
    modalAlertWithText: (text) => `*[role="dialog"] .pf-v5-c-alert:contains("${text}")`,
    currentWatchedImageRow: (name) =>
        `${selectors.currentWatchedImagesTable} tr:has(td:contains("${name}"))`,
    removeImageFromTableButton: (name) =>
        `${selectors.currentWatchedImagesTable} tr:has(td:contains("${name}")) button:contains("Remove watch")`,

    generateSbomModal: '*[role="dialog"]:contains("Generate Software Bill of Materials (SBOM)")',
};
