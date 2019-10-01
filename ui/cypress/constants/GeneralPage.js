const selectors = {
    navLinks: {
        first: 'nav.left-navigation li:first a',
        others: 'nav.left-navigation li:not(:first) a',
        list: 'nav.top-navigation li',
        apidocs: '[data-test-id="api-docs"]',
        apiDocsMenuLinks: '[data-test-id="api-docs-menu"] li'
    },
    sidePanel: '.navigation-panel'
};

export default selectors;
