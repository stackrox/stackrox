import { selectors } from '../../constants/PoliciesPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import DndSimulatorDataTransfer from '../../helpers/dndSimulatorDataTransfer';
import {
    visitPolicies,
    doPolicyRowAction,
    editFirstPolicyFromTable,
    cloneFirstPolicyFromTable,
    goToStep3,
} from '../../helpers/policies';
import { closeModalByButton } from '../../helpers/modal';
import { hasFeatureFlag } from '../../helpers/features';
import { getInputByLabel } from '../../helpers/formHelpers';

const dataTransfer = new DndSimulatorDataTransfer();

function dragFieldIntoSection(fieldSelector) {
    cy.get(fieldSelector)
        .trigger('mousedown', {
            which: 1,
        })
        .trigger('dragstart', {
            dataTransfer,
        })
        .trigger('drag');
    cy.get(selectors.step3.policySection.dropTarget)
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
}

function addPolicyFieldCard(index) {
    cy.get(selectors.step3.policyCriteria.key)
        .eq(index)
        .trigger('mousedown', { which: 1 })
        .trigger('dragstart', { dataTransfer })
        .trigger('drag');
    cy.get(selectors.step3.policySection.dropTarget)
        .trigger('dragover', { dataTransfer })
        .trigger('drop', { dataTransfer })
        .trigger('dragend', { dataTransfer })
        .trigger('mouseup', { which: 1 });
}

function clickPolicyKeyGroup(categoryName) {
    cy.get(
        `${selectors.step3.policyCriteria.keyGroup}:contains(${categoryName}) .pf-c-expandable-section__toggle`
    ).click();
}

function goToPoliciesAndCloneToStep3() {
    visitPolicies();
    cloneFirstPolicyFromTable();
    goToStep3();
}

function clearPolicyCriteriaCards() {
    // starting from clean slate
    cy.get(selectors.step3.policyCriteria.groupCards).then((cards) => {
        if (cards.length > 0) {
            cy.get(selectors.step3.policyCriteria.deleteBtn).eq(0).click();
        }
    });
}

describe('Policy wizard, Step 3 Policy Criteria', () => {
    withAuth();

    before(function () {
        if (hasFeatureFlag('ROX_POLICY_CRITERIA_MODAL')) {
            this.skip();
        }
    });

    it('should not allow user to edit policy criteria for default policies', () => {
        visitPolicies();
        editFirstPolicyFromTable();
        goToStep3();

        cy.get(selectors.step3.defaultPolicyAlert).should('exist');
        cy.get(selectors.step3.policyCriteria.value.numberInput).should('be.disabled');
        cy.get(selectors.step3.policySection.addBtn).should('not.exist');
    });

    it('should have nested policy field keys', () => {
        goToPoliciesAndCloneToStep3();

        cy.get(selectors.step3.policyCriteria.keyGroup).should((values) => {
            // before we began filtering what policy criteria were available,
            // there were 9 groups of criteria to count
            // after filtering for Lifecycle was added, the number of groups for a Deploy-only policy is 7
            const GROUPS_AVAILABLE_FOR_DEPLOY_POLICY = 7;
            expect(values).to.have.length(GROUPS_AVAILABLE_FOR_DEPLOY_POLICY);
        });

        cy.get(`${selectors.step3.policyCriteria.key}:first`).scrollIntoView().should('be.visible');
    });

    describe('Policy section', () => {
        it('should allow the user to add and delete a policy section card', () => {
            goToPoliciesAndCloneToStep3();

            // add policy section card
            cy.get(selectors.step3.policySection.cards).then((sections) => {
                cy.get(selectors.step3.policySection.addBtn).click();
                cy.get(selectors.step3.policySection.cards).then((newSections) => {
                    expect(newSections).to.have.length(sections.length + 1);
                });
            });
            cy.get(selectors.step3.policySection.orDivider).should('exist');

            // delete policy section card
            cy.get(selectors.step3.policySection.cards).then((sections) => {
                cy.get(selectors.step3.policySection.deleteBtn).first().click();
                cy.get(selectors.step3.policySection.cards).then((newSections) => {
                    expect(newSections).to.have.length(sections.length - 1);
                });
            });
        });

        it('should allow editing a policy section name and retain new name value', () => {
            goToPoliciesAndCloneToStep3();

            cy.get(selectors.step3.policySection.nameEditBtn).click();
            cy.get(selectors.step3.policySection.nameInput).clear().type('New Section');
            cy.get(selectors.step3.policySection.nameSaveBtn).click();
            cy.get(selectors.step3.policySection.name).contains('New Section');
        });

        it('should allow the user to add/delete a policy field card in the same policy section', () => {
            goToPoliciesAndCloneToStep3();

            // add policy field card
            cy.get(selectors.step3.policyCriteria.groupCards).then((cards) => {
                addPolicyFieldCard(0);
                cy.get(selectors.step3.policyCriteria.groupCards).then((newCards) => {
                    expect(newCards).to.have.length(cards.length + 1);
                });
            });

            // delete policy field card
            cy.get(selectors.step3.policyCriteria.groupCards).then((cards) => {
                cy.get(selectors.step3.policyCriteria.deleteBtn).eq(0).click();
                cy.get(selectors.step3.policyCriteria.groupCards).then((newCards) => {
                    expect(newCards).to.have.length(cards.length - 1);
                });
            });
        });

        it('should allow the user to add multiple non-duplicate policy field cards in the same policy section', () => {
            goToPoliciesAndCloneToStep3();

            cy.get(selectors.step3.policyCriteria.groupCards).then((cards) => {
                addPolicyFieldCard(0);
                addPolicyFieldCard(1);
                addPolicyFieldCard(2);
                cy.get(selectors.step3.policyCriteria.groupCards).then((newCards) => {
                    expect(newCards).to.have.length(cards.length + 3);
                });
            });
        });

        it('should not be able to add duplicate policy field cards in the same policy section', () => {
            goToPoliciesAndCloneToStep3();

            cy.get(selectors.step3.policyCriteria.groupCards).then((cards) => {
                addPolicyFieldCard(0);
                addPolicyFieldCard(0);
                cy.get(selectors.step3.policyCriteria.groupCards).then((newCards) => {
                    expect(newCards).to.have.length(cards.length + 1);
                });
            });
        });
    });

    describe('Policy field card', () => {
        describe('values', () => {
            it('should add/delete multiple field values for the same field if applicable', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                // add field values for Image Registry
                dragFieldIntoSection(
                    `${selectors.step3.policyCriteria.key}:contains('Image registry')`
                );
                cy.get(selectors.step3.policyCriteria.value.deleteBtn).should('not.exist');
                cy.get(selectors.step3.policyCriteria.value.addBtn).first().click();
                cy.get(selectors.step3.policyCriteria.value.textInput).then((inputs) => {
                    expect(inputs).to.have.length(2);
                    cy.get(selectors.step3.policyCriteria.value.deleteBtn).should('have.length', 2);
                });
                cy.get(selectors.step3.policyCriteria.booleanOperator).should('have.length', 1);
                cy.get(selectors.step3.policyCriteria.booleanOperator).should('not.be.disabled');

                // delete field value
                cy.get(selectors.step3.policyCriteria.value.deleteBtn).first().click();
                cy.get(selectors.step3.policyCriteria.value.textInput).then((inputs) => {
                    expect(inputs).to.have.length(1);
                    cy.get(selectors.step3.policyCriteria.value.deleteBtn).should('not.exist');
                    cy.get(selectors.step3.policyCriteria.booleanOperator).should('not.exist');
                });

                // TODO: (vjw, 2023-10-30) currently, this feature flag is only _adding_ another way to add policy criteria fields
                //       after adding fields has been thoroughly tested, this flag will indicate _whether_ to test the old way or the new way
                if (hasFeatureFlag('ROX_POLICY_CRITERIA_MODAL')) {
                    cy.get('.policy-section-card button:contains("Add policy field")').click();
                    cy.get('.pf-c-modal-box__title-text:contains("Add policy criteria field")');

                    // ensure closing modal with no actions
                    closeModalByButton('Cancel');

                    // now, add a field with modal
                    cy.get('.policy-section-card button:contains("Add policy field")').click();
                    cy.get(
                        'button.pf-c-tree-view__node:contains("Container configuration")'
                    ).click();
                    cy.get('button.pf-c-tree-view__node:contains("Environment variable")')
                        .click()
                        .should('have.class', 'pf-m-current');
                    cy.get('.pf-c-modal-box__footer button:contains("Add policy field")').click();

                    getInputByLabel('Key').clear().type('dev');
                    getInputByLabel('Value').clear().type('true');
                }
            });

            it('should not add multiple field values for the same field if not applicable', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                clickPolicyKeyGroup('Storage');
                dragFieldIntoSection(
                    `${selectors.step3.policyCriteria.key}:contains('Mounted volume writability')`
                );
                cy.get(selectors.step3.policyCriteria.value.radioGroup).should('exist');
                cy.get(selectors.step3.policyCriteria.value.addBtn).should('not.exist');
            });
        });

        describe('negation', () => {
            it('should negate field if applicable and change wording', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                dragFieldIntoSection(
                    `${selectors.step3.policyCriteria.key}:contains('Image registry')`
                );
                cy.get(selectors.step3.policyCriteria.value.negateCheckbox).should(
                    'not.be.checked'
                );
                cy.get(selectors.step3.policyCriteria.value.negateCheckbox).click();
                cy.get(selectors.step3.policyCriteria.value.negateCheckbox).should('be.checked');
                cy.get(selectors.step3.policyCriteria.groupCardTitle).first().contains('not');
            });

            it('should not show negate field if not applicable', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                clickPolicyKeyGroup('Storage');
                dragFieldIntoSection(
                    `${selectors.step3.policyCriteria.key}:contains('Mounted volume writability')`
                );
                cy.get(selectors.step3.policyCriteria.value.negateCheckbox).should('not.exist');
            });
        });

        describe('boolean operator', () => {
            it('should toggle AND/OR if applicable', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                dragFieldIntoSection(
                    `${selectors.step3.policyCriteria.key}:contains('Image registry')`
                );
                cy.get(selectors.step3.policyCriteria.value.addBtn).first().click();
                cy.get(selectors.step3.policyCriteria.booleanOperator).should('not.be.disabled');
                cy.get(selectors.step3.policyCriteria.booleanOperator).contains('or');
                cy.get(selectors.step3.policyCriteria.booleanOperator).click();
                cy.get(selectors.step3.policyCriteria.booleanOperator).contains('and');
            });

            it('should have AND/OR toggle disabled if not applicable', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                clickPolicyKeyGroup('Image contents');
                dragFieldIntoSection(`${selectors.step3.policyCriteria.key}:contains('Image age')`);
                cy.get(selectors.step3.policyCriteria.value.addBtn).first().click();
                cy.get(selectors.step3.policyCriteria.booleanOperator).should('be.disabled');
                cy.get(selectors.step3.policyCriteria.booleanOperator).contains('or');
            });
        });

        describe('input', () => {
            it('should populate boolean radio buttons w default value and respect changed values', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                clickPolicyKeyGroup('Image contents');
                dragFieldIntoSection(
                    `${selectors.step3.policyCriteria.key}:contains('Image scan status')`
                );
                cy.get(selectors.step3.policyCriteria.value.radioGroup).should('exist');
                cy.get(
                    `${selectors.step3.policyCriteria.value.radioGroupItem}:contains('Scanned') button`
                ).should('not.have.class', 'pf-m-selected');
                cy.get(
                    `${selectors.step3.policyCriteria.value.radioGroupItem}:contains('Not scanned') button`
                ).should('have.class', 'pf-m-selected');
                cy.get(
                    `${selectors.step3.policyCriteria.value.radioGroupItem}:contains('Scanned') button`
                ).click();
                cy.get(
                    `${selectors.step3.policyCriteria.value.radioGroupItem}:contains('Scanned') button`
                ).should('have.class', 'pf-m-selected');
                cy.get(
                    `${selectors.step3.policyCriteria.value.radioGroupItem}:contains('Not scanned') button`
                ).should('not.have.class', 'pf-m-selected');
            });

            it('should populate string radio buttons w default value and respect changed values', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                clickPolicyKeyGroup('Container configuration');
                dragFieldIntoSection(
                    `${selectors.step3.policyCriteria.key}:contains('Seccomp profile type')`
                );
                cy.get(selectors.step3.policyCriteria.value.radioGroupString).should('exist');
                cy.get(
                    `${selectors.step3.policyCriteria.value.radioGroupStringItem} button.pf-m-selected`
                ).should('have.length', 0);
                cy.get(
                    `${selectors.step3.policyCriteria.value.radioGroupStringItem}:contains('Unconfined') button`
                ).click();
                cy.get(
                    `${selectors.step3.policyCriteria.value.radioGroupStringItem}:contains('Unconfined') button`
                ).should('have.class', 'pf-m-selected');
                cy.get(
                    `${selectors.step3.policyCriteria.value.radioGroupStringItem} button.pf-m-selected`
                ).should('have.length', 1);
            });

            it('should populate text input and respect changed values', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                dragFieldIntoSection(
                    `${selectors.step3.policyCriteria.key}:contains('Image registry')`
                );
                cy.get(selectors.step3.policyCriteria.value.textInput).should('have.value', '');
                cy.get(selectors.step3.policyCriteria.value.textInput).type('test');
                cy.get(selectors.step3.policyCriteria.value.textInput).should('have.value', 'test');
            });

            it('should populate select dropdown and respect changed values', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                clickPolicyKeyGroup('Container configuration');
                dragFieldIntoSection(
                    `${selectors.step3.policyCriteria.key}:contains('Drop capabilities')`
                );
                cy.get(selectors.step3.policyCriteria.value.select).should('have.value', '');
                cy.get(selectors.step3.policyCriteria.value.select).click();
                cy.get(selectors.step3.policyCriteria.value.selectOption)
                    .first()
                    .then((option) => {
                        cy.wrap(option).click();
                        cy.get(selectors.step3.policyCriteria.value.select).contains(option.text());
                    });
            });

            it('should populate multiselect dropdown and respect changed values', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                clickPolicyKeyGroup('Storage');
                dragFieldIntoSection(
                    `${selectors.step3.policyCriteria.key}:contains('Mount propagation')`
                );
                cy.get(selectors.step3.policyCriteria.value.multiselect).should('have.value', '');
                cy.get(selectors.step3.policyCriteria.value.multiselect).click();
                cy.get(selectors.step3.policyCriteria.value.multiselectOption)
                    .first()
                    .then((option) => {
                        cy.wrap(option).click();
                        cy.get(selectors.step3.policyCriteria.value.multiselect).contains(
                            option.text()
                        );
                    });
            });

            it('should populate policy field input nested group and parse value string to object and respect changed values', () => {
                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();

                clickPolicyKeyGroup('Image contents');
                dragFieldIntoSection(`${selectors.step3.policyCriteria.key}:contains('CVSS')`);
                cy.get(selectors.step3.policyCriteria.value.select).should('have.value', '');
                cy.get(selectors.step3.policyCriteria.value.numberInput).should('have.value', '');
                cy.get(selectors.step3.policyCriteria.value.select).click();
                cy.get(selectors.step3.policyCriteria.value.selectOption)
                    .first()
                    .then((option) => {
                        cy.wrap(option).click();
                        cy.get(selectors.step3.policyCriteria.value.select).contains(option.text());
                    });
                cy.get(selectors.step3.policyCriteria.value.numberInput).type('10');
                cy.get(selectors.step3.policyCriteria.value.numberInput).should('have.value', '10');
            });
        });

        describe('table modal', () => {
            beforeEach(() => {
                cy.intercept('GET', api.integrations.signatureIntegrations, {
                    fixture: 'integrations/signatureIntegrations.json',
                }).as('getSignatureIntegrations');

                goToPoliciesAndCloneToStep3();
                clearPolicyCriteriaCards();
                dragFieldIntoSection(
                    `${selectors.step3.policyCriteria.key}:contains('Image signature')`
                );
                cy.wait('@getSignatureIntegrations');
            });

            it('should populate table modal select and respect changed values on save', () => {
                cy.get(selectors.step3.policyCriteria.value.tableModal.textInput).should(
                    'have.value',
                    'Add trusted image signers'
                );
                cy.get(selectors.step3.policyCriteria.value.tableModal.openButton).click();
                cy.get(selectors.step3.policyCriteria.value.tableModal.firstRowCheckbox).click();
                cy.get(selectors.step3.policyCriteria.value.tableModal.saveBtn).click();
                cy.get(selectors.step3.policyCriteria.value.tableModal.textInput).should(
                    'have.value',
                    'Selected 1 trusted image signer'
                );
            });

            it('should populate table modal select and not change values on cancel', () => {
                cy.get(selectors.step3.policyCriteria.value.tableModal.openButton).click();
                cy.get(selectors.step3.policyCriteria.value.tableModal.firstRowCheckbox).click();
                cy.get(selectors.step3.policyCriteria.value.tableModal.cancelBtn).click();
                cy.get(selectors.step3.policyCriteria.value.tableModal.textInput).should(
                    'have.value',
                    'Add trusted image signers'
                );
            });

            it('should go to link when table row is clicked', () => {
                cy.get(selectors.step3.policyCriteria.value.tableModal.openButton).click();
                cy.get(selectors.step3.policyCriteria.value.tableModal.firstRowName).click();
                cy.location('pathname').should('contain', 'signatureIntegrations');
            });
        });
    });

    describe('Existing values', () => {
        it('should populate boolean radio buttons', () => {
            visitPolicies();
            doPolicyRowAction(`${selectors.table.rows}:contains('root filesystem')`, 'Clone');
            goToStep3();
            cy.get(selectors.step3.policyCriteria.value.radioGroup).should('exist');
            cy.get(
                `${selectors.step3.policyCriteria.value.radioGroupItem}:contains('Writable') button`
            ).should('have.class', 'pf-m-selected');
            cy.get(
                `${selectors.step3.policyCriteria.value.radioGroupItem} button.pf-m-selected`
            ).should('have.length', 1);
        });

        it('should populate string radio buttons', () => {
            visitPolicies();
            doPolicyRowAction(`${selectors.table.rows}:contains('seccomp profile')`, 'Clone');
            goToStep3();
            cy.get(selectors.step3.policyCriteria.value.radioGroupString).should('exist');
            cy.get(
                `${selectors.step3.policyCriteria.value.radioGroupStringItem}:contains('Unconfined') button`
            ).should('have.class', 'pf-m-selected');
            cy.get(
                `${selectors.step3.policyCriteria.value.radioGroupStringItem} button.pf-m-selected`
            ).should('have.length', 1);
        });

        it('should populate text input', () => {
            visitPolicies();
            doPolicyRowAction(`${selectors.table.rows}:contains('Latest tag')`, 'Clone');
            goToStep3();
            cy.get(selectors.step3.policyCriteria.value.textInput).should('have.value', 'latest');
        });

        it('should populate select dropdown', () => {
            visitPolicies();
            doPolicyRowAction(`${selectors.table.rows}:contains('capability')`, 'Clone');
            goToStep3();
            cy.get(selectors.step3.policyCriteria.value.select).contains('SYS_ADMIN');
        });

        it('should populate multiselect dropdown', () => {
            visitPolicies();
            doPolicyRowAction(`${selectors.table.rows}:contains('mount propagation')`, 'Clone');
            goToStep3();
            cy.get(selectors.step3.policyCriteria.value.multiselect).contains('Bidirectional');
        });

        it('should populate policy field input nested group', () => {
            visitPolicies();
            doPolicyRowAction(`${selectors.table.rows}:contains('CVSS >= 6')`, 'Clone');
            goToStep3();
            cy.get(selectors.step3.policyCriteria.value.select).contains(
                'Is greater than or equal to'
            );
            cy.get(selectors.step3.policyCriteria.value.numberInput).should(
                'have.value',
                '6.000000'
            );
        });
    });
});
