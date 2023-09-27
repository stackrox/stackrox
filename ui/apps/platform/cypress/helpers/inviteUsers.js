import { accessModalSelectors } from '../integration/accessControl/accessControl.selectors';

export function checkInviteUsersModal() {
    // check that the modal opened
    cy.get(`${accessModalSelectors.title}:contains("Invite users")`);
    cy.get(`${accessModalSelectors.button}:contains("Invite users")`);

    // test closing the modal
    cy.get(`${accessModalSelectors.button}:contains("Cancel")`).click();
}
