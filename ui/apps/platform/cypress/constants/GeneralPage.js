const selectors = {
    navLinks: {
        first: 'ul.pf-v5-c-nav__list li:first a',
        others: 'ul.pf-v5-c-nav__list li:not(:first) a',
        apidocs: '[data-testid="API Reference"]',
    },
    leftNavLinks: 'nav.left-navigation li a',
    sidePanel: '.navigation-panel',
};

export default selectors;
