import * as api from '../../constants/apiEndpoints';
import { selectors } from '../../constants/PoliciesPage';
import withAuth from '../../helpers/basicAuth';
import { doPolicyRowAction, visitPolicies } from '../../helpers/policies';

describe('Import policy', () => {
    withAuth();

    it('should open and close dialog box', () => {
        visitPolicies();

        cy.get(selectors.table.importButton).click();

        cy.get(selectors.importUploadModal.titleText);
        cy.get(selectors.importUploadModal.beginButton).should('be.disabled');

        cy.get(selectors.importUploadModal.cancelButton).click();
        cy.get(selectors.importUploadModal.titleText).should('not.exist');
    });

    it('should import policy and then delete it', () => {
        visitPolicies();

        const fileName = 'policies/good_policy_to_import.json';
        cy.fixture(fileName).then((contents) => {
            const importedPolicyName = contents.policies[0].name;

            cy.get(`${selectors.table.policyLink}:contains("${importedPolicyName}")`).should(
                'not.exist'
            );

            cy.get(selectors.table.importButton).click();

            cy.get(selectors.importUploadModal.fileInput).selectFile(
                {
                    contents,
                    fileName,
                },
                { force: true } // because input element has display: none style
            );
            cy.get(
                `${selectors.importUploadModal.policyNames}:nth-child(1):contains("${importedPolicyName}")`
            );

            cy.intercept('POST', api.policies.import).as('importPolicy');
            cy.get(selectors.importUploadModal.beginButton).click();
            cy.wait('@importPolicy');

            cy.get(
                `${selectors.importSuccessModal.policyNames}:nth-child(1):contains("${importedPolicyName}")`
            );

            // After 3 seconds, success modal closes, and then table displays imported policy.
            cy.intercept('GET', `${api.policies.policies}?query=`).as('getPolicies');
            cy.wait('@getPolicies');
            cy.get(`${selectors.table.policyLink}:contains("${importedPolicyName}")`);

            const trSelector = `tbody tr:contains("${importedPolicyName}")`;
            doPolicyRowAction(trSelector, 'Delete policy');
            cy.get(selectors.confirmationModal.deleteButton).click();
            cy.get(`${selectors.toast.title}:contains("Successfully deleted policy")`);
        });
    });

    it('it should display options for policy which has duplicate name', () => {
        visitPolicies();

        cy.get(selectors.table.importButton).click();

        const contents = {
            policies: [
                {
                    name: 'Dupe Name Policy',
                },
            ],
        };
        cy.get(selectors.importUploadModal.fileInput).selectFile(
            {
                contents,
                fileName: 'dummy.json',
            },
            { force: true } // because input element has display: none style
        );

        const body = {
            responses: [
                {
                    succeeded: false,
                    policy: {
                        id: 'f09f8da1-6111-4ca0-8f49-294a76c65118',
                        name: 'Dupe Name Policy',
                        // Omit other policy properties.
                    },
                    errors: [
                        {
                            message: 'Could not add policy due to name validation',
                            type: 'duplicate_name',
                            duplicateName: 'Dupe Name Policy',
                        },
                    ],
                },
            ],
            allSucceeded: false,
        };
        cy.intercept('POST', api.policies.import, { body }).as('importPolicy');
        cy.get(selectors.importUploadModal.beginButton).click();
        cy.wait('@importPolicy');

        // Alert and disabled button.
        cy.get(selectors.importUploadModal.duplicateAlertTitle);
        cy.get(selectors.importUploadModal.duplicateNameSubstring);
        cy.get(selectors.importUploadModal.resumeButton).should('be.disabled');

        // Overwrite option enables the button.
        cy.get(selectors.importUploadModal.overwriteRadioLabel).click();
        cy.get(selectors.importUploadModal.resumeButton).should('be.enabled');

        // Rename option requires a new name to enable the button.
        cy.get(selectors.importUploadModal.renameRadioLabel).click();
        cy.get(selectors.importUploadModal.resumeButton).should('be.disabled');

        // Input a new name to enable the button (but cannot import the incomplete policy).
        cy.get(selectors.importUploadModal.renameInput).click();
        cy.get(selectors.importUploadModal.renameInput).type('A whole new world');
        cy.get(selectors.importUploadModal.resumeButton).should('be.enabled');
    });

    it('should display options for policy which has duplicate id', () => {
        visitPolicies();

        cy.get(selectors.table.importButton).click();

        const contents = {
            policies: [
                {
                    name: 'Dupe ID Policy',
                },
            ],
        };
        cy.get(selectors.importUploadModal.fileInput).selectFile(
            {
                contents,
                fileName: 'dummy.json',
            },
            { force: true } // because input element has display: none style
        );

        const body = {
            responses: [
                {
                    succeeded: false,
                    policy: {
                        id: 'f09f8da1-6111-4ca0-8f49-294a76c65117',
                        name: 'Fixable CVSS >= 9',
                        // other policy properties omitted from mock
                    },
                    errors: [
                        {
                            message:
                                'Policy Different than Fixable CVSS is >= 9 (f09f8da1-6111-4ca0-8f49-294a76c65117) cannot be added because it already exists',
                            type: 'duplicate_id',
                            duplicateName: 'Fixable CVSS >= 9',
                        },
                    ],
                },
            ],
            allSucceeded: false,
        };
        cy.intercept('POST', api.policies.import, { body }).as('importPolicy');
        cy.get(selectors.importUploadModal.beginButton).click();
        cy.wait('@importPolicy');

        // Alert and disabled button.
        cy.get(selectors.importUploadModal.duplicateAlertTitle);
        cy.get(selectors.importUploadModal.duplicateIdSubstring);
        cy.get(selectors.importUploadModal.resumeButton).should('be.disabled');

        // Overwrite option enables the button.
        cy.get(selectors.importUploadModal.overwriteRadioLabel).click();
        cy.get(selectors.importUploadModal.resumeButton).should('not.be.disabled');

        // Keep both option enables the button
        cy.get(selectors.importUploadModal.keepBothRadioLabel).click();
        cy.get(selectors.importUploadModal.resumeButton).should('not.be.disabled');
    });

    it('should display options for policy which has duplicate name and id', () => {
        visitPolicies();

        cy.get(selectors.table.importButton).click();

        const contents = {
            policies: [
                {
                    name: 'Dupe Name and Dupe ID Policy',
                },
            ],
        };
        cy.get(selectors.importUploadModal.fileInput).selectFile(
            {
                contents,
                fileName: 'dummy.json',
            },
            { force: true } // because input element has display: none style
        );

        const body = {
            responses: [
                {
                    succeeded: false,
                    policy: {
                        id: '8ac93556-4ad4-4220-a275-3f518db0ceb9',
                        name: 'Fixable CVSS >= 9',
                        // other policy properties omitted from mock
                    },
                    errors: [
                        {
                            message:
                                'Policy Fixable CVSS >= 9 (8ac93556-4ad4-4220-a275-3f518db0ceb9) cannot be added because it already exists',
                            type: 'duplicate_id',
                            duplicateName: 'Container using read-write root filesystem',
                        },
                        {
                            message: 'Could not add policy due to name validation',
                            type: 'duplicate_name',
                            duplicateName: 'Fixable CVSS >= 9',
                        },
                    ],
                },
            ],
            allSucceeded: false,
        };
        cy.intercept('POST', api.policies.import, { body }).as('importPolicy');
        cy.get(selectors.importUploadModal.beginButton).click();
        cy.wait('@importPolicy');

        // Alert and disabled button.
        cy.get(selectors.importUploadModal.duplicateAlertTitle);
        cy.get(selectors.importUploadModal.duplicateIdSubstring);
        cy.get(selectors.importUploadModal.duplicateNameSubstring);
        cy.get(selectors.importUploadModal.resumeButton).should('be.disabled');

        // Overwrite option enables the button.
        cy.get(selectors.importUploadModal.overwriteRadioLabel).click();
        cy.get(selectors.importUploadModal.resumeButton).should('be.enabled');

        // Rename option requires a new name to enable the button.
        cy.get(selectors.importUploadModal.renameRadioLabel).click();
        cy.get(selectors.importUploadModal.resumeButton).should('be.disabled');

        // Input a new name to enable the button (but cannot import the incomplete policy).
        cy.get(selectors.importUploadModal.renameInput).click();
        cy.get(selectors.importUploadModal.renameInput).type('Two are better than one');
        cy.get(selectors.importUploadModal.resumeButton).should('be.enabled');
    });
});
