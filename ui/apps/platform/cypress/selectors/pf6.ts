const actionsColumnSelectors = {
    kebabToggle: '.pf-v6-c-menu-toggle',
    menuListButton: '.pf-v6-c-menu__list button',
} as const;

const columnManagementSelectors = {
    columnManagementLabel:
        // data-ouia-component-id for these labels are in the format of "ColumnManagementModal-column-<index>-label" so
        // we match the start and end of the data-ouia-component-id string
        '[data-ouia-component-id^="ColumnManagementModal-column"][data-ouia-component-id$="label"] label',
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

const select = 'div[data-ouia-component-type="PF6/Select"]';
const selectSelectors = {
    select,
    selectItem: `${select} *[role="listbox"] li`,
} as const;

const tabsSelectors = {
    tab: '[data-ouia-component-type="PF6/Tab"]',
    tabButton: '[data-ouia-component-type="PF6/TabButton"]',
} as const;

export default {
    ...actionsColumnSelectors,
    ...columnManagementSelectors,
    ...dropdownSelectors,
    ...menuSelectors,
    ...navSelectors,
    ...pageHeaderSelectors,
    ...selectSelectors,
    ...tabsSelectors,
};
