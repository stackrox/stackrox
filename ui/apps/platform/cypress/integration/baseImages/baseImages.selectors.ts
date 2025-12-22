export const selectors = {
    // Page elements
    pageTitle: 'h1:contains("Base Images")',
    pageDescription: 'p:contains("Manage approved base images")',
    addButton: 'button:contains("Add base image")',

    // Table
    table: 'table',
    tableHeader: {
        baseImagePath: 'th:contains("Base image path")',
        addedBy: 'th:contains("Added by")',
    },
    tableRows: 'tbody tr',
    rowKebabButton: 'button[aria-label="Kebab toggle"]',
    removeAction: 'button:contains("Remove")',

    // Add modal
    addModal: {
        title: 'h2:contains("Add base image path")',
        input: 'input#baseImagePath',
        saveButton: 'button:contains("Save")',
        cancelButton: 'button:contains("Cancel")',
        successAlert: '.pf-v5-c-alert:contains("Base image successfully added")',
        errorAlert: '.pf-v5-c-alert:contains("Error adding base image")',
        validationError: '.pf-v5-c-helper-text__item.pf-m-error',
    },

    // Delete confirmation modal
    deleteModal: {
        title: 'h2:contains("Delete base image?")',
        confirmButton: '*[role="dialog"] button:contains("Delete")',
        cancelButton: '*[role="dialog"] button:contains("Cancel")',
        errorAlert: '.pf-v5-c-alert:contains("Error removing base image")',
    },
};
