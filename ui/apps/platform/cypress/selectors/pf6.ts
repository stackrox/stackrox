const actionsColumnSelectors = {
    kebabToggle: '.pf-v6-c-menu-toggle',
    menuListButton: '.pf-v6-c-menu__list button',
} as const;

const dropdownSelectors = {
    dropdown: 'div[data-ouia-component-type="PF6/Dropdown"]',
    dropdownItem: '*[data-ouia-component-type="PF6/DropdownItem"]',
} as const;

const menu = 'div[data-ouia-component-type="PF6/Menu"]';
const menuSelectors = {
    menu,
    menuToggle: `button[data-ouia-component-type="PF6/MenuToggle"]`,
    menuItem: `${menu} *[role="menuitem"]`,
} as const;

const navSelectors = {
    nav: 'nav[data-ouia-component-type="PF6/Nav"]',
    navExpandable: `li[data-ouia-component-type="PF6/NavExpandable"]`,
    navItem: `li[data-ouia-component-type="PF6/NavItem"]`,
} as const;
const pageHeaderSelectors = {
    pageHeaderTitle: '[data-ouia-component-id="PageHeader-title"]',
    pageHeaderSubtitle: '[data-ouia-component-id="PageHeader-subtitle"]',
} as const;

const tabsSelectors = {
    tab: '[data-ouia-component-type="PF6/Tab"]',
    tabButton: '[data-ouia-component-type="PF6/TabButton"]',
} as const;

export default {
    ...actionsColumnSelectors,
    ...dropdownSelectors,
    ...menuSelectors,
    ...navSelectors,
    ...pageHeaderSelectors,
    ...tabsSelectors,
};
