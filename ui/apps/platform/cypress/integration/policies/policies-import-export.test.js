import { selectors, url } from '../../constants/PoliciesPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { goToFirstPolicy, visitPolicies } from '../../helpers/policies';

describe('policy import and export', () => {
    withAuth();

    describe('policy export', () => {
        it('should start an API call to get the policy in the detail panel', () => {
            visitPolicies();
            goToFirstPolicy();

            cy.location('pathname').then((pathname) => {
                const segments = pathname.split('/');
                const policyId = segments[segments.length - 1];

                cy.intercept('POST', api.policies.export).as('policyExport');
                cy.get(selectors.singlePolicyExportButton).click();
                cy.wait('@policyExport')
                    .its('request.body')
                    .should('deep.equal', {
                        policyIds: [policyId],
                    });
            });
        });

        it('should display an error when the export fails', () => {
            visitPolicies();
            goToFirstPolicy();

            cy.intercept('POST', api.policies.export, {
                statusCode: 400,
                body: {
                    message: 'Some policies could not be retrieved.',
                },
            }).as('policyExport');
            cy.get(selectors.singlePolicyExportButton).click();
            cy.wait('@policyExport');

            cy.get(selectors.toast).should('contain', 'Could not export the policy');
        });
    });

    describe('policy import', () => {
        it('should open the import dialog when button is clicked', () => {
            visitPolicies();

            cy.get(selectors.importPolicyButton).click();
            cy.get(`${selectors.policyImportModal.content}:contains("JSON")`);
            cy.get(selectors.policyImportModal.uploadIcon);
            cy.get(selectors.policyImportModal.fileInput);
            cy.get(`${selectors.policyImportModal.confirm}:contains("Begin Import")`).should(
                'be.disabled'
            );

            cy.get(selectors.policyImportModal.cancel).click();
            cy.get(selectors.policyImportModal.content).should('not.exist');
        });

        it('should successfully import a policy', () => {
            visitPolicies();

            cy.get(selectors.importPolicyButton).click();

            const fileName = 'policies/good_policy_to_import.json';
            cy.fixture(fileName).then((json) => {
                const expectedPolicyName = json?.policies[0]?.name;
                const expectedPolicyId = json?.policies[0]?.id;

                cy.get(selectors.policyImportModal.fileInput).attachFile({
                    fileContent: json,
                    fileName,
                    mimeType: 'application/json',
                    encoding: 'utf8',
                });
                cy.get(`${selectors.policyImportModal.policyNames}:first`)
                    .invoke('text')
                    .then((policyText) => {
                        expect(policyText).to.equal(expectedPolicyName);
                    });

                cy.intercept('POST', api.policies.import).as('policyImport');
                cy.get(selectors.policyImportModal.confirm).click();
                cy.wait('@policyImport');

                cy.get(selectors.policyImportModal.successMessage);

                cy.location('pathname').should('eq', `${url}/${expectedPolicyId}`);
            });
        });

        it('should show error and handle resolution form when new policy has a duplicate name', () => {
            visitPolicies();

            const mockDupeNameResponse = {
                responses: [
                    {
                        succeeded: false,
                        policy: {
                            id: 'f09f8da1-6111-4ca0-8f49-294a76c65118',
                            name: 'Dupe Name Policy',
                            // other policy properties omitted from mock
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
            cy.intercept('POST', api.policies.import, {
                body: mockDupeNameResponse,
            }).as('dupeImportName');

            cy.get(selectors.importPolicyButton).click();

            const dummyJson = {
                policies: [
                    {
                        name: 'Dupe Name Policy',
                    },
                ],
            };
            cy.get(selectors.policyImportModal.fileInput).attachFile({
                fileContent: dummyJson,
                fileName: 'dummy.json',
                mimeType: 'application/json',
                encoding: 'utf8',
            });
            cy.get(selectors.policyImportModal.confirm).click();

            cy.wait('@dupeImportName');

            // check error state
            cy.get(selectors.policyImportModal.dupeNameMessage);
            cy.get(selectors.policyImportModal.confirm).should('be.disabled');

            // first, ensure there is an overwrite option
            cy.get(selectors.policyImportModal.overwriteRadioLabel).click();
            cy.get(selectors.policyImportModal.confirm).should('be.enabled');

            // next, ensure there is a rename option, and that it requires more info than just clicking
            cy.get(selectors.policyImportModal.renameRadioLabel).click();
            cy.get(selectors.policyImportModal.confirm).should('be.disabled');

            // finally, give a new name, and ensure we can again submit the policy
            cy.get(selectors.policyImportModal.newNameInputLabel).click().type('A whole new world');
            cy.get(selectors.policyImportModal.confirm).should('be.enabled');
        });

        it('should show error and handle resolution form when new policy has a duplicate ID', () => {
            visitPolicies();

            const mockDupeNameResponse = {
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
            cy.intercept('POST', api.policies.import, {
                body: mockDupeNameResponse,
            }).as('dupeImportId');

            cy.get(selectors.importPolicyButton).click();

            const dummyJson = {
                policies: [
                    {
                        name: 'Dupe ID Policy',
                    },
                ],
            };
            cy.get(selectors.policyImportModal.fileInput).attachFile({
                fileContent: dummyJson,
                fileName: 'dummy.json',
                mimeType: 'application/json',
                encoding: 'utf8',
            });
            cy.get(selectors.policyImportModal.confirm).click();

            cy.wait('@dupeImportId');

            // check error state
            cy.get(selectors.policyImportModal.dupeIdMessage);
            cy.get(selectors.policyImportModal.confirm).should('be.disabled');

            // first, ensure there is an overwrite option
            cy.get(selectors.policyImportModal.overwriteRadioLabel).click();
            cy.get(selectors.policyImportModal.confirm).should('not.be.disabled');

            // finally, ensure there is a "keep both" option, and ensure we can again submit the policy
            cy.get(selectors.policyImportModal.keepBothRadioLabel).click();
            cy.get(selectors.policyImportModal.confirm).should('not.be.disabled');
        });

        it('should show error and handle resolution form when new policy has both duplicate name and duplicate ID', () => {
            visitPolicies();

            const mockDupeNameResponse = {
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
            cy.intercept('POST', api.policies.import, {
                body: mockDupeNameResponse,
            }).as('dupeImportNameAndId');

            cy.get(selectors.importPolicyButton).click();

            const dummyJson = {
                policies: [
                    {
                        name: 'Dupe Name and Dupe ID Policy',
                    },
                ],
            };
            cy.get(selectors.policyImportModal.fileInput).attachFile({
                fileContent: dummyJson,
                fileName: 'dummy.json',
                mimeType: 'application/json',
                encoding: 'utf8',
            });
            cy.get(selectors.policyImportModal.confirm).click();

            cy.wait('@dupeImportNameAndId');

            // check error state
            cy.get(selectors.policyImportModal.dupeNameMessage);
            cy.get(selectors.policyImportModal.dupeIdMessage);
            cy.get(selectors.policyImportModal.confirm).should('be.disabled');

            // first, ensure there is an overwrite option
            cy.get(selectors.policyImportModal.overwriteRadioLabel).click();
            cy.get(selectors.policyImportModal.confirm).should('be.enabled');

            // next, ensure there is a rename option, and that it requires more info than just clicking
            cy.get(selectors.policyImportModal.renameRadioLabel).click();
            cy.get(selectors.policyImportModal.confirm).should('be.disabled');

            // finally, give a new name, and ensure we can again submit the policy
            cy.get(selectors.policyImportModal.newNameInputLabel)
                .click()
                .type('A policy by any other name would smell just as sweet');
            cy.get(selectors.policyImportModal.confirm).should('be.enabled');
        });
    });
});
