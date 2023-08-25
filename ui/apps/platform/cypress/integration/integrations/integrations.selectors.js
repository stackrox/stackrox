export const selectors = {
    breadcrumbItem: '.pf-c-breadcrumb__item',
    tableRowNameLink: 'tbody td a', // TODO td[data-label="Name"] would be even better, but no dataLabel prop yet
    buttons: {
        test: 'button:contains("Test")',
        save: 'button:contains("Save")',
        generate: 'button:contains("Generate")',
        back: 'button:contains("Back")',
    },
};
