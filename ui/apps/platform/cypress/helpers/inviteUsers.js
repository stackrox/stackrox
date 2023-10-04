import { accessModalSelectors } from '../integration/accessControl/accessControl.selectors';
import { getInputByLabel, getSelectButtonByLabel, getSelectOption } from './formHelpers';

export function checkInviteUsersModal() {
    // check that the modal opened
    cy.get(`${accessModalSelectors.title}:contains("Invite users")`);
    cy.get(`${accessModalSelectors.button}:contains("Invite users")`);

    // check emails field
    getInputByLabel('Emails').click().type('scooby.doo@redhat.com').blur();

    // select auth provider
    getSelectButtonByLabel('Provider').click();
    getSelectOption('auth-provider-1').click();

    // check role field
    getSelectButtonByLabel('Role').click();
    cy.get(`.pf-c-select__menu-item`).should('have.length', 7);
    getSelectOption('Network Graph Viewer').click();

    // test closing the modal
    cy.get(`${accessModalSelectors.button}:contains("Cancel")`).click();
}
