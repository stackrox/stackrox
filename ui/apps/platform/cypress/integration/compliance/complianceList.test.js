import withAuth from '../../helpers/basicAuth';
import searchSelectors from '../../selectors/search';

import {
    assertComplianceEntityPage,
    interactAndWaitForComplianceEntities,
    interactAndWaitForComplianceEntityInSidePanel,
    triggerScan,
    visitComplianceEntities,
    visitComplianceStandard,
} from './Compliance.helpers';
import { selectors } from './Compliance.selectors';

describe('Compliance entities list', () => {
    withAuth();

    it('should filter namespaces table with passing controls', () => {
        triggerScan(); // in case complianceDashboard.test.js is skipped
        visitComplianceEntities('namespaces');

        interactAndWaitForComplianceEntities(() => {
            cy.get(searchSelectors.input).type('Compliance State:');
            cy.get(searchSelectors.input).type('{enter}');
            cy.get(searchSelectors.input).type('Pass');
            cy.get(searchSelectors.input).type('{enter}');
        }, 'namespaces');
        cy.get('.rt-tbody .rt-tr').should('not.exist');
        cy.get('[data-testid="panel-header"]').should('contain', '0 namespaces');
    });

    it('should filter namespaces table with failing controls', () => {
        visitComplianceEntities('namespaces');

        interactAndWaitForComplianceEntities(() => {
            cy.get(searchSelectors.input).type('Compliance State:');
            cy.get(searchSelectors.input).type('{enter}');
            cy.get(searchSelectors.input).type('Fail');
            cy.get(searchSelectors.input).type('{enter}');
        }, 'namespaces');
        cy.get('.rt-tbody .rt-tr');
        cy.get('[data-testid="panel-header"]').should('contain', 'namespace');
    });

    it('should open/close side panel when clicking on a table row', () => {
        visitComplianceEntities('clusters');

        cy.get(selectors.list.table.firstRowName)
            .invoke('text')
            .then((name) => {
                cy.get('[data-testid="panel-header"]').should('contain', 'cluster');
                cy.get(selectors.list.table.firstRow).click();
                cy.get('[data-testid="side-panel"]').should('exist');
                cy.get('[data-testid="side-panel-header"]').contains(name);
                cy.get('[data-testid="side-panel"] [aria-label="Close"]').click();
                cy.get('[data-testid="side-panel"]').should('not.exist');
                cy.get('[data-testid="panel-header"]').should('contain', 'cluster');
            });
    });

    it('should link to entity page when clicking on side panel header', () => {
        visitComplianceEntities('clusters');

        cy.get(selectors.list.table.firstRowName)
            .invoke('text')
            .then((name) => {
                interactAndWaitForComplianceEntityInSidePanel(() => {
                    cy.get(selectors.list.table.firstRow).click();
                }, 'clusters');
                cy.get('[data-testid="side-panel-header"]').contains(name);
                cy.get('[data-testid="side-panel-header"]').click();
                assertComplianceEntityPage('clusters');
            });
    });

    it('should be sorted by version in standards list', () => {
        visitComplianceStandard('CIS Kubernetes v1.5');

        cy.get(selectors.list.table.firstRowName)
            .invoke('text')
            .then((text1) => {
                cy.get(selectors.list.table.secondRowName)
                    .invoke('text')
                    .then((text2) => {
                        const arr1 = text1.split(' ')[0];
                        const controlArr1 = arr1.split('.');
                        const arr2 = text2.split(' ')[0];
                        const controlArr2 = arr2.split('.');
                        expect(parseInt(controlArr1[0], 10)).to.be.at.most(
                            parseInt(controlArr2[0], 10)
                        );
                        if (controlArr1[1] && controlArr2[1]) {
                            expect(parseInt(controlArr1[1], 10)).to.be.at.most(
                                parseInt(controlArr2[1], 10)
                            );
                        }
                    });
            });
    });

    it('should collapse/open table grouping', () => {
        visitComplianceStandard('PCI DSS 3.2.1');

        cy.get(selectors.list.table.firstTableGroup).should('be.visible');
        cy.get(selectors.list.table.firstGroup).click();
        cy.get(selectors.list.table.firstTableGroup).should('not.be.visible');
    });
});
