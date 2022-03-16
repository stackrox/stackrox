const selectors = {
    navLinks: {
        first: 'ul.pf-c-nav__list li:first a',
        others: 'ul.pf-c-nav__list li:not(:first) a',
        apidocs: '[data-testid="API Reference"]',
    },
    leftNavLinks: 'nav.left-navigation li a',
    sidePanel: '.navigation-panel',
    errorBoundary: '[data-testid="error-boundary"]',
};

export default selectors;
