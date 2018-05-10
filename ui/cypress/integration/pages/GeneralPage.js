const selectors = {
    navLinks: {
        first: 'nav.left-navigation li:first a',
        others: 'nav.left-navigation li:not(:first) a',
        list: 'nav.top-navigation li',
        compliance: 'nav.left-navigation li:contains("Compliance")'
    },
    sidePanel: '.navigation-panel'
};

export default selectors;
