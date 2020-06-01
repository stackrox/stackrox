import { selectors, text, url } from '../constants/PoliciesPage';
import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';
import DndSimulatorDataTransfer from '../helpers/dndSimulatorDataTransfer';
import checkFeatureFlag from '../helpers/features';

describe('Policies page', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.fixture('search/metadataOptions.json').as('metadataOptionsJson');
        cy.route('GET', api.search.options, '@metadataOptionsJson').as('metadataOptions');
        cy.visit(url);
        cy.wait('@metadataOptions');
    });

    const openActionMenu = () => {
        cy.get(selectors.actionMenuBtn).click();
    };

    const addPolicy = () => {
        cy.get(selectors.newPolicyButton).click();
    };

    const editPolicy = () => {
        cy.get(selectors.editPolicyButton).click();
    };

    const closePolicySidePanel = () => {
        cy.get(selectors.cancelButton).click();
    };

    const savePolicy = () => {
        // Next will dryrun and show the policy effects preview.
        cy.route('POST', api.policies.dryrun).as('dryrunPolicy');
        cy.get(selectors.nextButton).click();
        cy.wait('@dryrunPolicy');
        // Next will now take you to the enforcement page.
        cy.get(selectors.nextButton).click();
        // Save will PUT the policy (assuming it is not new), then GET it.
        cy.route('PUT', api.policies.policy).as('savePolicy');
        cy.route('GET', api.policies.policy).as('getPolicy');
        cy.get(selectors.savePolicyButton).click();
        cy.wait('@savePolicy');
        cy.wait('@getPolicy');
    };

    describe('basic tests', () => {
        it('should navigate using the left nav', () => {
            cy.visit('/');
            cy.get(selectors.configure).click();
            cy.get(selectors.navLink).click({ force: true });
            cy.location('pathname').should('eq', url);
        });

        it('should display and send a query using the search input', () => {
            cy.route('/v1/policies?query=Category:DevOps Best Practices').as('newSearchQuery');
            cy.get(selectors.searchInput).type('Category:{enter}');
            cy.get(selectors.searchInput).type('DevOps Best Practices{enter}');
            cy.wait('@newSearchQuery');
            cy.get(selectors.searchInput).type('{backspace}{backspace}');
            cy.route('/v1/policies?query=Cluster:remote').as('newSearchQuery');
            cy.get(selectors.searchInput).type('Cluster:{enter}');
            cy.get(selectors.searchInput).type('remote{enter}');
            cy.wait('@newSearchQuery');
        });

        it('should show the required "*" next to the required fields', () => {
            addPolicy();
            cy.get(selectors.form.required).eq(0).prev().should('have.text', 'Name');
            cy.get(selectors.form.required).eq(1).prev().should('have.text', 'Severity');
            cy.get(selectors.form.required).eq(2).prev().should('have.text', 'Lifecycle Stages');
            cy.get(selectors.form.required).eq(3).prev().should('have.text', 'Categories');
        });

        it('should have selected item in nav bar', () => {
            cy.get(selectors.configure).should('have.class', 'bg-primary-700');
        });

        it('should open side panel and check for the policy name', () => {
            cy.get(selectors.tableFirstRowName)
                .invoke('text')
                .then((name) => {
                    cy.get(selectors.tableFirstRow).click({ force: true });
                    cy.get(selectors.sidePanel).should('exist');
                    cy.get(selectors.sidePanelHeader).contains(name);
                });
        });

        it('should allow updating policy name', () => {
            const updatePolicyName = (typeStr) => {
                editPolicy();
                cy.get(selectors.tableContainer).should('have.class', 'pointer-events-none');
                cy.get(selectors.form.nameInput).type(typeStr);
                savePolicy();
            };
            const secretSuffix = ':secretSuffix:';
            const deleteSuffix = '{backspace}'.repeat(secretSuffix.length);

            cy.get(selectors.tableFirstRow).click({ force: true });
            updatePolicyName(secretSuffix);
            cy.get(`.rt-tr:contains("${secretSuffix}")`);
            updatePolicyName(deleteSuffix); // revert back
        });

        it('should not allow getting a dry run when creating a policy with a duplicate name', () => {
            addPolicy();
            cy.get(selectors.form.nameInput).type(text.policyLatestTagName);
            cy.get(selectors.nextButton).click();
            cy.get(selectors.nextButton).should('exist');
        });

        it('should show dry run loading screen before showing dry run results', () => {
            cy.get(selectors.tableFirstRow).click({ force: true });
            cy.get(selectors.editPolicyButton).click();
            cy.get(selectors.nextButton).click();
            cy.get(selectors.policyPreview.loading).should('exist');
            closePolicySidePanel();
        });

        it('should open the preview panel to view policy dry run', () => {
            cy.get(selectors.tableFirstRow).click({ force: true });
            cy.get(selectors.editPolicyButton).click();
            cy.get(selectors.nextButton).click();

            cy.get(selectors.policyPreview.loading).should('exist');
            cy.wait(2000);

            cy.get('.warn-message').should('exist');
            cy.get('.alert-preview').should('exist');
            cy.get('.whitelist-exclusions').should('exist');
            closePolicySidePanel();
        });

        it('should open the panel to create a new policy', () => {
            addPolicy();
            cy.get(selectors.nextButton).should('exist');
        });

        it('should show a specific message when editing a policy with "enabled" value as "no"', () => {
            cy.get(selectors.policies.disabledPolicyImage).click({ force: true });
            editPolicy();
            cy.get(selectors.nextButton).click();
            cy.get(selectors.policyPreview.message).should('have.text', text.policyPreview.message);
        });

        // TODO(ROX-1580): Re-enable this test.
        xit('should de-highlight a row on panel close', () => {
            // Select a row.
            cy.route('GET', api.policies.policy).as('getPolicy');
            cy.get(selectors.policies.scanImage).click({ force: true });
            cy.wait('@getPolicy'); // Wait for the panel to be loaded before closing.

            // Check that the row is active and highlighted
            cy.get(selectors.policies.scanImage).should('have.class', 'row-active');

            // Close the side panel.
            closePolicySidePanel();

            // Check that it is no longer active and highlighted.
            cy.get(selectors.policies.scanImage).should('not.have.class', 'row-active');
        });

        it('should have details panel open on page refresh', () => {
            // Select a row.
            cy.get(selectors.policies.scanImage).click({ force: true });

            // Reload the page with that row's id in the URL.
            cy.get(selectors.policyDetailsPanel.idValueDiv)
                .invoke('text')
                .then((idValue) => {
                    cy.visit(url.concat('/', idValue));
                });

            // Check that the side panel is open.
            cy.get(selectors.cancelButton).should('exist');
        });

        it('should show Add Capabilities value in edit mode', () => {
            cy.get(selectors.policies.addCapabilities).click({ force: true });
            editPolicy();
            cy.get(selectors.form.selectValue).contains('CAP_SYS_ADMIN');
            closePolicySidePanel();
        });

        // TODO: (ROX-3373) make this test work with updated babel and cypress versions
        it.skip('should allow disable/enable policy from the policies table', () => {
            // initialize to have enabled policy
            cy.get(selectors.enableDisableIcon)
                .first()
                .then((icon) => {
                    if (!icon.hasClass(selectors.enabledIconColor))
                        cy.get(selectors.hoverActionButtons).first().click({ force: true });
                });

            // force click the first enable/disable button on the first row
            cy.get(selectors.hoverActionButtons).first().click({ force: true });

            cy.get(selectors.enableDisableIcon)
                .first()
                .should('not.have.class', selectors.enabledIconColor);
            cy.get(selectors.tableFirstRow).click({ force: true });
            cy.get(selectors.policyDetailsPanel.enabledValueDiv).should('contain', 'No');

            cy.get(selectors.hoverActionButtons).first().click({ force: true }); // enable policy
            cy.get(selectors.policyDetailsPanel.enabledValueDiv).should('contain', 'Yes');
            cy.get(selectors.enableDisableIcon)
                .first()
                .should('have.class', selectors.enabledIconColor);
        });

        it('should show action menu when the checkboxes are chosen', () => {
            cy.get(selectors.reassessAllButton).should('be.visible');
            cy.get(selectors.newPolicyButton).should('be.visible');
            cy.get(selectors.checkboxes).eq(1).click({ force: true });
            cy.get(selectors.actionMenuBtn).should('be.visible');
            openActionMenu();
            cy.get(selectors.actionMenu).contains('Delete');
            cy.get(selectors.actionMenu).contains('Enable');
            cy.get(selectors.actionMenu).contains('Disable');
            cy.get(selectors.reassessAllButton).should('not.be.visible');
            cy.get(selectors.newPolicyButton).should('not.be.visible');
        });

        it('should delete a policy when the hover delete policy clicked', () => {
            cy.get(selectors.tableFirstRow).click({ force: true });
            cy.get(selectors.sidePanel).should('exist');
            cy.get(selectors.tableFirstRowName)
                .invoke('text')
                .then((policyName) => {
                    cy.get(selectors.tableFirstRow).should('contain', policyName);
                    cy.get(selectors.hoverActionButtons).eq(1).click({ force: true });
                    cy.get(selectors.tableFirstRow).should('not.contain', policyName);
                    cy.get(selectors.tableFirstRow).click({ force: true });
                    cy.get(selectors.sidePanel).should('exist');
                    cy.get(selectors.sidePanelHeader).should('not.have.text', policyName);
                });
        });

        it('should allow creating new categories and saving them (ROX-1409)', () => {
            const categoryName = 'ROX-1409-test-category';
            cy.get(selectors.tableFirstRow).click({ force: true });
            editPolicy();
            cy.get(selectors.categoriesField.input).type(`${categoryName}{enter}`);
            savePolicy();

            // now edit same policy, the previous category should exist in the list
            editPolicy();
            cy.get(
                `${selectors.categoriesField.valueContainer} > div:contains(${categoryName}) > div.react-select__multi-value__remove`
            ).click(); // remove it
            savePolicy();
        });
    });

    describe('policy import and export', () => {
        before(function beforeHook() {
            // skip the whole suite if policy import/export isn't enabled
            if (checkFeatureFlag('ROX_POLICY_IMPORT_EXPORT', false)) {
                this.skip();
            }
        });

        describe('policy export', () => {
            it('should start an API call to get the policy in the detail panel', () => {
                cy.route({
                    method: 'POST',
                    url: 'v1/policies/export',
                }).as('policyExport');

                cy.get(selectors.tableFirstRow).click();

                cy.url().then((href) => {
                    const segments = href.split('/');
                    const policyId = segments[segments.length - 1];
                    cy.get(selectors.singlePolicyExportButton).click();

                    cy.wait('@policyExport')
                        .its('request.body')
                        .should('deep.equal', {
                            policyIds: [policyId],
                        });
                });
            });

            it('should display an error when the export fails', () => {
                cy.route({
                    method: 'POST',
                    url: 'v1/policies/export',
                    status: 400,
                    response: {
                        message: 'Some policies could not be retrieved.',
                    },
                }).as('policyExport');

                cy.get(selectors.tableFirstRow).click();
                cy.get(selectors.singlePolicyExportButton).click();

                cy.wait('@policyExport');

                cy.get(selectors.toast).contains('Could not export the policy');
            });
        });

        describe('policy import', () => {
            it('should open the import dialog when button is clicked', () => {
                cy.get(selectors.importPolicyButton).click();

                cy.get(`${selectors.policyImportModal.content}:contains("JSON")`);
                cy.get(selectors.policyImportModal.uploadIcon);
                cy.get(selectors.policyImportModal.fileInput);
                cy.get(selectors.policyImportModal.confirm)
                    .should('be.disabled')
                    .invoke('text')
                    .then((btnText) => {
                        expect(btnText).to.contain('Import');
                    });

                cy.get(selectors.policyImportModal.cancel).click();
                cy.get(selectors.policyImportModal.content).should('not.exist');
            });

            it('should successfully import a policy', () => {
                cy.get(selectors.importPolicyButton).click();

                const fileName = 'policies/good_policy_to_import.json';
                cy.fixture(fileName).then((json) => {
                    const expectedPolicyName = json?.policies[0]?.name;
                    const expectedPolicyId = json?.policies[0]?.id;

                    // due to way Cypress handles JSON fixtures, we have to use this workaround to handle JSON file upload
                    //   https://github.com/abramenal/cypress-file-upload/issues/175#issue-586835434
                    const fileContent = JSON.stringify(json);
                    cy.get(selectors.policyImportModal.fileInput).upload({
                        fileContent,
                        fileName,
                        mimeType: 'application/json',
                        encoding: 'utf8',
                    });
                    cy.get(`${selectors.policyImportModal.policyNames}:first`)
                        .invoke('text')
                        .then((policyText) => {
                            expect(policyText).to.equal(expectedPolicyName);
                        });

                    cy.get(selectors.policyImportModal.confirm).click();

                    cy.get(selectors.policyImportModal.successMessage);

                    cy.location('pathname').should('eq', `${url}/${expectedPolicyId}`);
                });
            });

            it('should show error and handle resolution form when new policy has a duplicate name', () => {
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
                cy.route({
                    method: 'POST',
                    url: 'v1/policies/import',
                    response: mockDupeNameResponse,
                }).as('dupeImportName');

                cy.get(selectors.importPolicyButton).click();

                const dummyJson = {
                    policies: [
                        {
                            name: 'Dupe Name Policy',
                        },
                    ],
                };
                const fileContent = JSON.stringify(dummyJson);
                cy.get(selectors.policyImportModal.fileInput).upload({
                    fileContent,
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
                cy.get(selectors.policyImportModal.confirm).should('not.be.disabled');

                // next, ensure there is a rename option, and that it requires more info than just clicking
                cy.get(selectors.policyImportModal.renameRadioLabel).click();
                cy.get(selectors.policyImportModal.confirm).should('be.disabled');

                // finally, give a new name, and ensure we can again submit the policy
                cy.get(selectors.policyImportModal.newNameInputLabel)
                    .click()
                    .type('A whole new world');
                cy.get(selectors.policyImportModal.confirm).should('not.be.disabled');
            });

            it('should show error and handle resolution form when new policy has a duplicate ID', () => {
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
                cy.route({
                    method: 'POST',
                    url: 'v1/policies/import',
                    response: mockDupeNameResponse,
                }).as('dupeImportId');

                cy.get(selectors.importPolicyButton).click();

                const dummyJson = {
                    policies: [
                        {
                            name: 'Dupe ID Policy',
                        },
                    ],
                };
                const fileContent = JSON.stringify(dummyJson);
                cy.get(selectors.policyImportModal.fileInput).upload({
                    fileContent,
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
                cy.route({
                    method: 'POST',
                    url: 'v1/policies/import',
                    response: mockDupeNameResponse,
                }).as('dupeImportNameAndId');

                cy.get(selectors.importPolicyButton).click();

                const dummyJson = {
                    policies: [
                        {
                            name: 'Dupe Name and Dupe ID Policy',
                        },
                    ],
                };
                const fileContent = JSON.stringify(dummyJson);
                cy.get(selectors.policyImportModal.fileInput).upload({
                    fileContent,
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
                cy.get(selectors.policyImportModal.confirm).should('not.be.disabled');

                // next, ensure there is a rename option, and that it requires more info than just clicking
                cy.get(selectors.policyImportModal.renameRadioLabel).click();
                cy.get(selectors.policyImportModal.confirm).should('be.disabled');

                // finally, give a new name, and ensure we can again submit the policy
                cy.get(selectors.policyImportModal.newNameInputLabel)
                    .click()
                    .type('A policy by any other name would smell just as sweet');
                cy.get(selectors.policyImportModal.confirm).should('not.be.disabled');
            });
        });
    });

    describe('pre-Boolean Policy Logic tests (deprecated)', () => {
        before(function beforeHook() {
            // skip the whole suite if BPL is enabled
            if (checkFeatureFlag('ROX_BOOLEAN_POLICY_LOGIC', true)) {
                this.skip();
            }
        });

        it('should allow floats for numeric configuration fields, like CVSS', () => {
            cy.get(selectors.tableFirstRow).click({ force: true });

            editPolicy();
            cy.get(selectors.configurationField.selectArrow).first().click();
            cy.get(selectors.configurationField.options).contains('CVSS').click();
            cy.get(selectors.configurationField.numericInput).last().type(2.2);

            savePolicy();
        });

        it('should allow updating image fields in a policy', () => {
            cy.get(selectors.policies.scanImage).click({ force: true });
            editPolicy();
            // cy.get(selectors.form.select).select('Image Registry');

            cy.get(selectors.configurationField.selectArrow).first().click();
            cy.get(selectors.configurationField.options).contains('Image Registry').click();

            cy.get(selectors.imageRegistry.input).type('docker.io');
            savePolicy();
            cy.get(selectors.imageRegistry.value).should(
                'have.text',
                'Alert on any image using any tag from registry docker.io'
            );
            editPolicy();
            cy.get(selectors.imageRegistry.deleteButton).click();
            savePolicy();
        });

        it('should allow updating days since image scanned in a policy', () => {
            cy.get(selectors.policies.scanImage).click({ force: true });
            editPolicy();
            cy.get(selectors.configurationField.selectArrow).first().click();
            cy.get(selectors.configurationField.options)
                .contains('Days since image was last scanned')
                .click();

            cy.get(selectors.scanAgeDays.input).type('50');
            savePolicy();
            cy.get(selectors.scanAgeDays.value).should('have.text', '50 Days ago');
            editPolicy();
            cy.get(selectors.scanAgeDays.deleteButton).click();
            savePolicy();
            cy.get(selectors.scanAgeDays.value).should('not.exist');
        });
    });

    describe('Boolean Policy Logic Section', () => {
        before(function beforeHook() {
            // skip the whole suite if BPL isn't enabled
            if (checkFeatureFlag('ROX_BOOLEAN_POLICY_LOGIC', false)) {
                this.skip();
            }
        });

        const dataTransfer = new DndSimulatorDataTransfer();

        const dragFieldIntoSection = (fieldSelector) => {
            cy.get(fieldSelector)
                .trigger('mousedown', {
                    which: 1,
                })
                .trigger('dragstart', {
                    dataTransfer,
                })
                .trigger('drag');
            cy.get(selectors.booleanPolicySection.policySectionDropTarget)
                .trigger('dragover', {
                    dataTransfer,
                })
                .trigger('drop', {
                    dataTransfer,
                })
                .trigger('dragend', {
                    dataTransfer,
                })
                .trigger('mouseup', {
                    which: 1,
                });
        };

        const addPolicySection = () => {
            cy.get(selectors.booleanPolicySection.addPolicySectionBtn).click();
        };

        const addPolicyFieldCard = (index) => {
            cy.get(selectors.booleanPolicySection.policyKey)
                .eq(index)
                .trigger('mousedown', { which: 1 })
                .trigger('dragstart', { dataTransfer })
                .trigger('drag');
            cy.get(selectors.booleanPolicySection.policySectionDropTarget)
                .trigger('dragover', { dataTransfer })
                .trigger('drop', { dataTransfer })
                .trigger('dragend', { dataTransfer })
                .trigger('mouseup', { which: 1 });
        };

        describe('Single Policy Field Card', () => {
            beforeEach(() => {
                addPolicy();
                addPolicySection();
            });
            it('should add multiple Field Values for the same Field with an AND/OR operator between them when (+) is clicked', () => {
                // to mock BPL policy here, but for now
                addPolicyFieldCard(0);
                cy.get(selectors.booleanPolicySection.addPolicyFieldValueBtn).click();
                cy.get(selectors.booleanPolicySection.policyFieldValue).should((values) => {
                    expect(values).to.have.length(2);
                });
            });

            it('should allow floats for CPU and CVSS configuration fields', () => {
                // first, select a CPU field
                dragFieldIntoSection(
                    `${selectors.booleanPolicySection.policyKey}:contains("Container CPU Request")`
                );

                cy.get(selectors.booleanPolicySection.form.selectArrow).first().click();
                cy.get(
                    `${selectors.booleanPolicySection.form.selectOption}:contains("Is equal to")`
                ).click();
                cy.get(selectors.booleanPolicySection.form.numericInput).click().type(2.2);

                // second, select CVSS field
                dragFieldIntoSection(
                    `${selectors.booleanPolicySection.policyKey}:contains("CVSS")`
                );

                cy.get(selectors.booleanPolicySection.form.selectArrow).last().click();
                cy.get(
                    `${selectors.booleanPolicySection.form.selectOption}:contains("Is greater than or equal to")`
                ).click();
                cy.get(`${selectors.booleanPolicySection.form.numericInput}:last`)
                    .click()
                    .type(7.5);
            });

            it('should allow updating image fields in a policy', () => {
                cy.get(selectors.policies.scanImage).click({
                    force: true,
                });
                editPolicy();

                // first, drag in an image field
                dragFieldIntoSection(
                    `${selectors.booleanPolicySection.policyKey}:contains("Image Registry")`
                );

                // second, add a value to it
                cy.get(`${selectors.booleanPolicySection.form.textInput}:last`)
                    .click()
                    .type('docker.io');
                savePolicy();

                // third, check that the new field and its value saved successfully
                cy.get(`${selectors.booleanPolicySection.policyFieldCard}:last`).should(
                    'have.text',
                    'Image pulled from registry:'
                );
                cy.get(`${selectors.booleanPolicySection.policyFieldCard}:last input`).should(
                    'have.value',
                    'docker.io'
                );

                // clean up, by removing the field we just added
                editPolicy();
                cy.get(`${selectors.booleanPolicySection.removePolicyFieldBtn}:last`).click();
                savePolicy();
            });

            it('should allow updating days since image scanned in a policy', () => {
                cy.get(selectors.policies.scanImage).click({
                    force: true,
                });
                editPolicy();

                // first, drag in an image scan age field
                dragFieldIntoSection(
                    `${selectors.booleanPolicySection.policyKey}:contains("Image Scan Age")`
                );

                // second, add a value to it
                cy.get(`${selectors.booleanPolicySection.form.numericInput}:last`)
                    .click()
                    .type('50');
                savePolicy();

                // third, check that the new field and its value saved successfully
                cy.get(`${selectors.booleanPolicySection.policyFieldCard}:last`).should(
                    'have.text',
                    'Minimum days since last image scan:'
                );
                cy.get(`${selectors.booleanPolicySection.policyFieldCard}:last input`).should(
                    'have.value',
                    '50'
                );

                // clean up, by removing the field we just added
                editPolicy();
                cy.get(`${selectors.booleanPolicySection.removePolicyFieldBtn}:last`).click();
                savePolicy();
            });

            it('should not allow multiple Policy Field Values for boolean Policy Fields', () => {
                // to mock BPL policy here, but for now
                dragFieldIntoSection(
                    `${selectors.booleanPolicySection.policyKey}:contains("Root")`
                );

                cy.get(selectors.booleanPolicySection.addPolicyFieldValueBtn).should('not.exist');
            });

            it('should delete only the selected Policy Value from a Policy Field', () => {
                // to mock BPL policy here, but for now
                addPolicyFieldCard(0);
                cy.get(selectors.booleanPolicySection.addPolicyFieldValueBtn).click();
                cy.get(selectors.booleanPolicySection.removePolicyFieldValueBtn).eq(0).click();
                cy.get(selectors.booleanPolicySection.policyFieldValue).then((values) => {
                    expect(values).to.have.length(1);
                });
                cy.get(selectors.booleanPolicySection.removePolicyFieldValueBtn).should(
                    'not.exist'
                );
            });
        });

        describe('Single Policy Section', () => {
            it('should populate a default Value input in a new Policy Section when dragging a Field Key', () => {
                addPolicy();
                addPolicySection();
                cy.get(selectors.booleanPolicySection.policyFieldCard).should('not.exist');
                addPolicyFieldCard(0);
                cy.get(selectors.booleanPolicySection.policyFieldCard).should('exist');
                cy.get(selectors.booleanPolicySection.policyFieldValue).should('exist');
                cy.get(
                    `${selectors.booleanPolicySection.policySection} ${selectors.booleanPolicySection.andOrOperator}`
                ).should('contain', 'AND');
            });

            it('should AND the dragged Field when dragging a Field Key to a Policy Section that already has a Field ', () => {
                // to mock BPL policy here, but for now
                addPolicy();
                addPolicySection();
                addPolicyFieldCard(0);
                addPolicyFieldCard(1);
                cy.get(selectors.booleanPolicySection.policyFieldValue).should((values) => {
                    expect(values).to.have.length(2);
                });

                cy.get(
                    `${selectors.booleanPolicySection.policySection} ${selectors.booleanPolicySection.andOrOperator}`
                ).should((andOrOperators) => {
                    expect(andOrOperators).to.have.length(2);
                });
            });

            it('should delete the Field from the Policy Section', () => {
                // to mock BPL policy here, but for now
                addPolicy();
                addPolicySection();
                addPolicyFieldCard(0);
                cy.get(selectors.booleanPolicySection.policyFieldCard).should('exist');
                cy.get(selectors.booleanPolicySection.removePolicyFieldBtn).click();
                cy.get(selectors.booleanPolicySection.policyFieldCard).should('not.exist');
            });

            it('should not allow dragging a duplicate Field Key in the same Policy Section', () => {
                // to mock BPL policy here, but for now
                addPolicy();
                addPolicySection();
                addPolicyFieldCard(0);
                addPolicyFieldCard(0);
                cy.get(selectors.booleanPolicySection.policyFieldValue).should((values) => {
                    expect(values).to.have.length(1);
                });
            });
        });

        // describe('AND operator', () => {
        //     it('should toggle to OR when AND is clicked if the Policy Field is can be ANDed', () => {

        //     })

        //     it('should be disabled if the Policy Field cannot be ORed', () => {

        //     });
        // });

        // describe('OR operator', () => {
        //     it('should toggle to AND when OR is clicked if the Policy Field can be ORed', () => {

        //     })

        //     it('should be disabled if the Policy Field cannot be ANDed', () => {

        //     });
        // })

        // describe('Policy Field Card NOT toggle', () => {
        //     it('should negate the Policy Field Card when the toggle is clicked & should show negated text', () => {

        //     })

        //     it('should be disabled if the Policy Field cannot be negated', () => {

        //     })
        // })

        describe('Multiple Policy Sections', () => {
            it('should add a Policy Section with a pre-populated Policy Section header', () => {
                addPolicy();
                addPolicySection();
                cy.get(selectors.booleanPolicySection.policySection).then(() => {
                    cy.get(selectors.booleanPolicySection.sectionHeader.text)
                        .invoke('text')
                        .then((headerText) => {
                            expect(headerText).to.equal('Policy Section 1');
                        });
                });
            });

            it('should delete a Policy Section', () => {
                addPolicy();
                addPolicySection();
                cy.get(selectors.booleanPolicySection.removePolicySectionBtn).click();
                cy.get(selectors.booleanPolicySection.policySection).should('not.exist');
            });

            it('should edit the Policy Section header name', () => {
                addPolicy();
                addPolicySection();
                cy.get(selectors.booleanPolicySection.sectionHeader.editBtn).click();
                const newHeaderText = 'new policy section';
                cy.get(selectors.booleanPolicySection.sectionHeader.input).type(
                    `{selectall}${newHeaderText}`
                );
                cy.get(selectors.booleanPolicySection.sectionHeader.confirmBtn).click();
                cy.get(selectors.booleanPolicySection.sectionHeader.text)
                    .invoke('text')
                    .then((headerText) => {
                        expect(headerText).to.equal(newHeaderText);
                    });
            });

            it('should read in data properly when provided', () => {});
        });
        // describe('Policy Field Keys', () => {
        //     it('should be grouped into catgories', () => {});
        //     it('should collapse categories when clicking the carrot', () => {});
        // });
    });
});
