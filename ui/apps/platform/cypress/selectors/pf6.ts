const navSelectors = {
    nav: 'nav[data-ouia-component-type="PF6/Nav"]',
    navExpandable: `li[data-ouia-component-type="PF6/NavExpandable"]`,
    navItem: `li[data-ouia-component-type="PF6/NavItem"]`,
} as const;

const menu = 'div[data-ouia-component-type="PF6/Menu"]';
const menuSelectors = {
    menu,
    menuToggle: `button[data-ouia-component-type="PF6/MenuToggle"]`,
    menuItem: `${menu} *[role="menuitem"]`,
} as const;

export default {
    ...navSelectors,
    ...menuSelectors,
};
