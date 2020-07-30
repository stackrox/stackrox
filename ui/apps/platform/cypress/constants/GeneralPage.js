const selectors = {
    navLinks: {
        first: 'nav.left-navigation li:first a',
        others: 'nav.left-navigation li:not(:first) a',
        list: 'nav.top-navigation li',
        apidocs: '[data-testid="API Reference"]',
    },
    sidePanel: '.navigation-panel',
    errorBoundary: '[data-testid="error-boundary"]',
};

export default selectors;
