import { accessModalSelectors } from '../integration/accessControl/accessControl.selectors';
import { getInputByLabel, getSelectButtonByLabel, getSelectOption } from './formHelpers';

/**
 * resuable function to check the form elements in the Invite Users modal
 *
 * assumes the following:
 * - an auth provider named "auth-provider-1" is available
 * - the standard built-in roles are available
 * - the modal is open
 */
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
    cy.get(`.pf-v5-c-menu .pf-v5-c-menu__list .pf-v5-c-menu__item`).should('have.length', 7);
    getSelectOption('Network Graph Viewer').click();
}
