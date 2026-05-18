import withAuth from '../../../helpers/basicAuth';
import { readFileFromDownloads } from '../../../helpers/file';
import { getDescriptionListGroup, getInputByLabel } from '../../../helpers/formHelpers';
import pf6 from '../../../selectors/pf6';

import { visitClusters } from '../Clusters.helpers';

import { cleanupClusterRegistrationSecretsWithName } from './ClusterRegistrationSecrets.helpers';

describe('Cluster registration secrets', () => {
    withAuth();

    const testCrsName = 'CYPRESS-E2E-TEST-CRS';

    beforeEach(() => {
        cleanupClusterRegistrationSecretsWithName(testCrsName);
    });

    afterEach(() => {
        cleanupClusterRegistrationSecretsWithName(testCrsName);
    });

    it('should create a new Cluster registration secret and then view and delete', () => {
        visitClusters();

        cy.get('a:contains("Cluster registration secrets")').click();

        const crsLinkInTableSelector = `td[data-label="Name"] a:contains("${testCrsName}")`;

        // Verify that old CRS from Cypress tests are not present
        cy.get('table');
        cy.get(crsLinkInTableSelector).should('not.exist');

        // Create a new CRS and verify that the YAML is downloaded and the secret appears in the table
        cy.get(`${pf6.button}:contains("Create cluster registration secret")`).click();

        cy.get('button:contains("Download")').should('be.disabled');
        getInputByLabel('Name').clear().type(testCrsName);

        // Set validity to "By date" and enter a date 30 days from now
        cy.get('label:contains("By date")').click();
        const futureDate = new Date();
        futureDate.setDate(futureDate.getDate() + 30);
        const formattedDate = futureDate.toISOString().split('T')[0]; // Extract YYYY-MM-DD
        cy.get('input[placeholder="YYYY-MM-DD"]').type(formattedDate);

        getInputByLabel('Max registrations').clear().type('5');

        cy.get('button:contains("Download")').click();

        readFileFromDownloads(`${testCrsName}-cluster-registration-secret.yaml`).should('exist');

        // Click through to the detail page and verify fields
        cy.get('table');
        cy.get(crsLinkInTableSelector).click();

        getDescriptionListGroup('Max registrations', '5').should('exist');
        getDescriptionListGroup('Expires at', formattedDate).should('exist');

        // Go back to the table
        cy.go('back');

        // Revoke the secret
        cy.get('table');
        cy.get(`tr:has(${crsLinkInTableSelector}) button[aria-label="Kebab toggle"]`).click();
        // The Revoke button in the table menu
        cy.get('ul[role="menu"] button:contains("Revoke cluster registration secret")').click();
        // The Revoke confirmation button in the modal
        cy.get('div[role="dialog"] button:contains("Revoke cluster registration secret")').click();

        cy.get('table');
        cy.get(crsLinkInTableSelector).should('not.exist');
    });
});
