const navExpandable = 'ul.pf-c-nav__list li.pf-c-nav__item.pf-m-expandable button';

const navigation = {
    navLinks: '.pf-c-nav > ul.pf-c-nav__list > li > a',
    navExpandable,
    navExpandablePlatformConfiguration: `${navExpandable}:contains("Platform Configuration")`,
    navExpandableVulnerabilityManagement: `${navExpandable}:contains("Vulnerability Management")`,
    nestedNavLinks: '.pf-c-nav__subnav ul.pf-c-nav__list li a',
    leftNavBar: 'nav.left-navigation li a',
    navPanel: '.navigation-panel ul li a',
    topNavBar: '[data-testid="top-nav-bar"]',
    summaryCounts: '[data-testid="top-nav-bar"] [data-testid="summary-tile-count"]',
};

export default navigation;
