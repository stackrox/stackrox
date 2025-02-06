import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { selectFilteredWorkflowView, visitViolations } from './Violations.helpers';

describe('Violations - Filtered Workflow Views', () => {
    withAuth();

    before(function () {
        if (
            !hasFeatureFlag('ROX_PLATFORM_COMPONENTS') ||
            !hasFeatureFlag('ROX_PLATFORM_CVE_SPLIT')
        ) {
            this.skip();
        }
    });

    it('should filter the violations table when the "Applications view" is selected', () => {
        visitViolations();

        cy.intercept('GET', '/v1/alerts?query=*').as('getViolations');

        // should filter using the correct values for the "Applications view"
        cy.wait('@getViolations').then((interception) => {
            const queryString = interception.request.query.query;

            expect(queryString).to.contain('Entity Type:DEPLOYMENT');
            expect(queryString).to.contain('Platform Component:false');
        });
    });

    it('should filter the violations table when the "Platform view" is selected', () => {
        visitViolations();

        selectFilteredWorkflowView('Platform');

        cy.intercept('GET', '/v1/alerts?query=*').as('getViolations');

        // should filter using the correct values for the "Platform view"
        cy.wait('@getViolations').then((interception) => {
            const queryString = interception.request.query.query;

            expect(queryString).to.contain('Entity Type:DEPLOYMENT');
            expect(queryString).to.contain('Platform Component:true');
        });
    });

    it('should filter the violations table when the "Full view" is selected', () => {
        visitViolations();

        selectFilteredWorkflowView('All Violations');

        cy.intercept('GET', '/v1/alerts?query=*').as('getViolations');

        // should filter using the correct values for the "Full view"
        cy.wait('@getViolations').then((interception) => {
            const queryString = interception.request.query.query;

            expect(queryString).to.not.contain('Entity Type:DEPLOYMENT');
            expect(queryString).to.not.contain('Platform Component:true');
            expect(queryString).to.not.contain('Platform Component:false');
        });
    });
});
