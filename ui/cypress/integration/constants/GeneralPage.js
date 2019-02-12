const selectors = {
    navLinks: {
        first: 'nav.left-navigation li:first a',
        others: 'nav.left-navigation li:not(:first) a',
        list: 'nav.top-navigation li',
        apidocs: '[data-test-id="nav-footer"] a'
    },
    sidePanel: '.navigation-panel'
};

export default selectors;
