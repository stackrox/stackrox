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

const tabsSelectors = {
    tab: '[data-ouia-component-type="PF6/Tab"]',
    tabButton: '[data-ouia-component-type="PF6/TabButton"]',
} as const;

export default {
    ...dropdownSelectors,
    ...menuSelectors,
    ...navSelectors,
    ...tabsSelectors,
};
