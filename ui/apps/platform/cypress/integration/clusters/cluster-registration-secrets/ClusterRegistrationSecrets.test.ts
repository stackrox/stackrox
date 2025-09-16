import withAuth from '../../../helpers/basicAuth';
import { readFileFromDownloads } from '../../../helpers/file';
import { getInputByLabel } from '../../../helpers/formHelpers';

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
        // TODO dv 2025-04-01
        // From a user's point of view, this is a "button". It would be nice if we had a unified way to
        // click a button without needing to know the underlying HTML element used by the component.
        cy.get('a:contains("Create cluster registration secret")').click();

        cy.get('button:contains("Download")').should('be.disabled');
        getInputByLabel('Name').clear().type(testCrsName);
        cy.get('button:contains("Download")').click();

        readFileFromDownloads(`${testCrsName}-cluster-registration-secret.yaml`).should('exist');

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
