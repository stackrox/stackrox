import { selectors, text } from '../../constants/PoliciesPage';
import withAuth from '../../helpers/basicAuth';
import DndSimulatorDataTransfer from '../../helpers/dndSimulatorDataTransfer';
import {
    addPolicySection,
    editPolicy,
    goToFirstPolicy,
    goToNamedPolicy,
    goToNewPolicyCriteria,
    goToNewPolicySummary,
    goToNextWizardStage,
    goToPrevWizardStage,
    savePolicy,
    visitPolicies,
} from '../../helpers/policies';

describe('Boolean Policy Logic Section', () => {
    withAuth();

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
    }

    function addPolicyFieldCard(index) {
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
    }

    function clickPolicyKeyGroup(categoryName) {
        cy.get(
            `${selectors.booleanPolicySection.policyKeyGroup}:contains(${categoryName}) ${selectors.booleanPolicySection.collapsibleBtn}`
        ).click();
    }

    describe('Single Policy Field Card 1', () => {
        it('should add multiple Field Values for the same Field with an AND/OR operator between them when (+) is clicked', () => {
            goToNewPolicyCriteria();
            addPolicySection();

            // to mock BPL policy here, but for now
            addPolicyFieldCard(0);
            cy.get(selectors.booleanPolicySection.addPolicyFieldValueBtn).click();
            cy.get(selectors.booleanPolicySection.policyFieldValue).should((values) => {
                expect(values).to.have.length(2);
            });
        });

        it('should allow floats for CPU configuration field', () => {
            goToNewPolicyCriteria();
            addPolicySection();

            // unfurl Container Configuration policy key group
            clickPolicyKeyGroup('Container configuration');
            // first, select a CPU field
            dragFieldIntoSection(
                `${selectors.booleanPolicySection.policyKey}:contains("Container CPU request")`
            );

            cy.get(selectors.booleanPolicySection.form.selectArrow).first().click();
            cy.get(
                `${selectors.booleanPolicySection.form.selectOption}:contains("Is equal to")`
            ).click();
            cy.get(selectors.booleanPolicySection.form.numericInput).click().type(2.2);
        });

        it('should allow floats for CVSS configuration field', () => {
            goToNewPolicyCriteria();
            addPolicySection();

            // unfurl Image Contents policy field key group
            clickPolicyKeyGroup('Image contents');
            // second, select CVSS field
            dragFieldIntoSection(`${selectors.booleanPolicySection.policyKey}:contains("CVSS")`);

            cy.get(selectors.booleanPolicySection.form.selectArrow).last().click();
            cy.get(
                `${selectors.booleanPolicySection.form.selectOption}:contains("Is greater than or equal to")`
            ).click();
            cy.get(`${selectors.booleanPolicySection.form.numericInput}:last`).click().type(7.5);
        });

        it('should not allow multiple Policy Field Values for boolean Policy Fields', () => {
            goToNewPolicyCriteria();
            addPolicySection();

            // unfurl Container Configuration policy key group
            clickPolicyKeyGroup('Container configuration');
            // to mock BPL policy here, but for now
            dragFieldIntoSection(`${selectors.booleanPolicySection.policyKey}:contains("root")`);

            cy.get(selectors.booleanPolicySection.addPolicyFieldValueBtn).should('not.exist');
        });

        it('should delete only the selected Policy Value from a Policy Field', () => {
            goToNewPolicyCriteria();
            addPolicySection();

            // to mock BPL policy here, but for now
            addPolicyFieldCard(0);
            cy.get(selectors.booleanPolicySection.addPolicyFieldValueBtn).click();
            cy.get(selectors.booleanPolicySection.removePolicyFieldValueBtn).eq(0).click();
            cy.get(selectors.booleanPolicySection.policyFieldValue).then((values) => {
                expect(values).to.have.length(1);
            });
            cy.get(selectors.booleanPolicySection.removePolicyFieldValueBtn).should('not.exist');
        });
    });

    // TODO: for release 65, re-enable these tests after UI to handle read-only flags is merged. ROX-7768: Enable once clone operation is fixed.
    describe.skip('Single Policy Field Card 2', () => {
        it('should allow updating days since image scanned in a policy', () => {
            visitPolicies();
            goToNamedPolicy(text.scanImage);
            editPolicy();
            goToNextWizardStage();

            // unfurl Image Contents Policy Key Group
            clickPolicyKeyGroup('Image contents');
            // first, drag in an image scan age field
            dragFieldIntoSection(
                `${selectors.booleanPolicySection.policyKey}:contains("Image scan age")`
            );

            // second, add a value to it
            cy.get(`${selectors.booleanPolicySection.form.numericInput}:last`).click().type('50');
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
            goToNextWizardStage();
            cy.get(`${selectors.booleanPolicySection.removePolicyFieldBtn}:last`).click();
            savePolicy();
        });

        it('should allow updating image fields in a policy', () => {
            visitPolicies();
            goToNamedPolicy(text.scanImage);
            editPolicy();
            goToNextWizardStage();

            // first, drag in an image field
            dragFieldIntoSection(
                `${selectors.booleanPolicySection.policyKey}:contains("Image registry")`
            );

            // second, add a value to it
            cy.get(`${selectors.booleanPolicySection.form.textInput}:last`)
                .click()
                .clear()
                .type('docker.io');
            savePolicy();

            // third, check that the new field and its value saved successfully
            cy.get(`${selectors.booleanPolicySection.policyFieldCard}:last`).should(
                'contain.text',
                'Image pulled from registry:'
            );
            cy.get(`${selectors.booleanPolicySection.policyFieldCard}:last input`).should(
                'have.value',
                'docker.io'
            );

            // clean up, by removing the field we just added
            editPolicy();
            goToNextWizardStage();
            cy.get(`${selectors.booleanPolicySection.removePolicyFieldBtn}:last`).click();
            savePolicy();
        });
    });

    describe('Single Policy Section', () => {
        it('should populate a default Value input in a new Policy Section when dragging a Field Key', () => {
            goToNewPolicyCriteria();
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
            goToNewPolicyCriteria();
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
            goToNewPolicyCriteria();
            addPolicySection();

            addPolicyFieldCard(0);
            cy.get(selectors.booleanPolicySection.policyFieldCard).should('exist');
            cy.get(selectors.booleanPolicySection.removePolicyFieldBtn).click();
            cy.get(selectors.booleanPolicySection.policyFieldCard).should('not.exist');
        });

        it('should not allow dragging a duplicate Field Key in the same Policy Section', () => {
            goToNewPolicyCriteria();
            addPolicySection();

            addPolicyFieldCard(0);
            addPolicyFieldCard(0);
            cy.get(selectors.booleanPolicySection.policyFieldValue).should((values) => {
                expect(values).to.have.length(1);
            });
        });
    });

    describe('Boolean operator', () => {
        it('should toggle to AND when OR is clicked if the Policy Field can be ANDed', () => {
            goToNewPolicyCriteria();
            addPolicySection();

            addPolicyFieldCard(0);
            cy.get(selectors.booleanPolicySection.addPolicyFieldValueBtn).click();
            const policyFieldCardAndOrOperator = `${selectors.booleanPolicySection.policyFieldCard} ${selectors.booleanPolicySection.andOrOperator}`;
            cy.get(policyFieldCardAndOrOperator).should('contain', 'OR');
            cy.get(policyFieldCardAndOrOperator).click();
            cy.get(policyFieldCardAndOrOperator).should('contain', 'AND');
            cy.get(policyFieldCardAndOrOperator).click();
            cy.get(policyFieldCardAndOrOperator).should('contain', 'OR');
        });

        it('should be disabled if the Policy Field cannot be ANDed', () => {
            goToNewPolicyCriteria();
            addPolicySection();

            // unfurl Image Contents policy key group
            clickPolicyKeyGroup('Image contents');
            dragFieldIntoSection(
                `${selectors.booleanPolicySection.policyKey}:contains("Image age")`
            );
            cy.get(selectors.booleanPolicySection.addPolicyFieldValueBtn).click();
            const policyFieldCardAndOrOperator = `${selectors.booleanPolicySection.policyFieldCard} ${selectors.booleanPolicySection.andOrOperator}`;
            cy.get(policyFieldCardAndOrOperator).should('contain', 'OR');
            cy.get(policyFieldCardAndOrOperator).click();
            cy.get(policyFieldCardAndOrOperator).should('contain', 'OR');
        });
    });

    describe('Policy Field Card NOT toggle', () => {
        it('should negate the Policy Field Card when the toggle is clicked & should show negated text', () => {
            goToNewPolicyCriteria();
            addPolicySection();

            addPolicyFieldCard(0);
            cy.get(selectors.booleanPolicySection.policyFieldCard).should(
                'contain',
                'Image pulled from registry'
            );
            cy.get(selectors.booleanPolicySection.notToggle).click();
            cy.get(selectors.booleanPolicySection.policyFieldCard).should(
                'contain',
                'Image not pulled from registry'
            );
        });

        it('should not exist if the Policy Field cannot be negated', () => {
            goToNewPolicyCriteria();
            addPolicySection();

            // unfurl Image Contents policy key group
            clickPolicyKeyGroup('Image contents');
            dragFieldIntoSection(
                `${selectors.booleanPolicySection.policyKey}:contains("Image age")`
            );
            cy.get(selectors.booleanPolicySection.policyFieldCard).should(
                'contain',
                'Minimum days since image was built'
            );
            cy.get(selectors.booleanPolicySection.notToggle).should('not.exist');
        });
    });

    describe('Policy Field Keys', () => {
        it('should be grouped into categories', () => {
            goToNewPolicyCriteria();

            cy.get(selectors.booleanPolicySection.policyKeyGroupBtn).should((values) => {
                expect(values).to.have.length(9);
            });
        });

        it('should filter keys based on Event Source value', () => {
            goToNewPolicySummary();

            cy.get(selectors.lifecycleStageField.input).type(`Runtime{enter}`);
            cy.get(selectors.eventSourceField.selectArrow).click();
            cy.get(`${selectors.eventSourceField.options}:contains("Audit Log")`).click();
            goToNextWizardStage();
            cy.get(selectors.booleanPolicySection.policyKeyGroupBtn).should((values) => {
                expect(values).to.have.length(1);
            });
        });

        it('should collapse categories when clicking the carrot', () => {
            goToNewPolicyCriteria();

            cy.get(`${selectors.booleanPolicySection.policyKey}:first`)
                .scrollIntoView()
                .should('be.visible');
            cy.get(`${selectors.booleanPolicySection.policyKeyGroupBtn}:first`).click();
            cy.get(`${selectors.booleanPolicySection.policyKeyGroupContent}:first`).should(
                'have.class',
                'overflow-hidden'
            );
        });

        it('should have categories collapsed by default if not first group', () => {
            goToNewPolicyCriteria();

            cy.get(`${selectors.booleanPolicySection.policyKeyGroupContent}:first`)
                .scrollIntoView()
                .should('be.visible');
            cy.get(`${selectors.booleanPolicySection.policyKeyGroupContent}:last`)
                .scrollIntoView()
                .should('have.class', 'overflow-hidden');
        });
    });

    describe('Multiple Policy Sections', () => {
        it('should add a Policy Section with a pre-populated Policy Section header', () => {
            goToNewPolicyCriteria();
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
            goToNewPolicyCriteria();
            addPolicySection();

            cy.get(selectors.booleanPolicySection.removePolicySectionBtn).click();
            cy.get(selectors.booleanPolicySection.policySection).should('not.exist');
        });

        it('should edit the Policy Section header name', () => {
            goToNewPolicyCriteria();
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
    });

    describe('Data consistency', () => {
        it('should read in data properly when provided', () => {
            visitPolicies();
            goToNamedPolicy(text.scanImage);

            cy.get(selectors.booleanPolicySection.policySection).scrollIntoView().should('exist');
            cy.get(selectors.booleanPolicySection.sectionHeader.text).should('exist');
            cy.get(selectors.booleanPolicySection.policyFieldCard).should(
                'contain',
                'Minimum days since image was built'
            );
            cy.get(`${selectors.booleanPolicySection.policyFieldValue} input`).should(
                'be.disabled'
            );
        });

        it('should keep same form values from edit details stage to edit criteria stage and back', () => {
            visitPolicies();
            goToFirstPolicy();
            editPolicy();

            cy.get(selectors.form.nameInput).type('1234');
            goToNextWizardStage();
            goToPrevWizardStage();
            cy.get(selectors.form.nameInput).should('contain.value', '1234');
        });
    });
});
