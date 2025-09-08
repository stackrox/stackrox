import pf6 from '../selectors/pf6';

export function selectProject(project: string) {
    cy.get(`.co-namespace-bar ${pf6.menuToggle}`).click();
    cy.get(`.co-namespace-bar ${pf6.menuItem}`)
        .contains(new RegExp(`^${project}$`, 'i'))
        .click();
}
