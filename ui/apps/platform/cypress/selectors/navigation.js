const navExpandable = 'ul.pf-v5-c-nav__list li.pf-v5-c-nav__item.pf-m-expandable button';

const navigation = {
    navLinks: '.pf-v5-c-nav > ul.pf-v5-c-nav__list > li > a',
    navExpandable,
    navExpandablePlatformConfiguration: `${navExpandable}:contains("Platform Configuration")`,
    navExpandableVulnerabilityManagement: `${navExpandable}:contains("Vulnerability Management")`,
    nestedNavLinks: '.pf-v5-c-nav__subnav ul.pf-v5-c-nav__list li a',
    leftNavBar: 'nav.left-navigation li a',
    navPanel: '.navigation-panel ul li a',
    topNavBar: '[data-testid="top-nav-bar"]',
    summaryCounts: '[data-testid="top-nav-bar"] [data-testid="summary-tile-count"]',
};

export default navigation;
