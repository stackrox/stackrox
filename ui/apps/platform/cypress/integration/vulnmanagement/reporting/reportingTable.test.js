import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';

import {
    visitVulnerabilityReportingFromLeftNav,
    visitVulnerabilityReportingWithFixture,
} from './reporting.helpers';

describe('Vulnerability Management Reporting table', () => {
    withAuth();

    before(function () {
        if (hasFeatureFlag('ROX_VULN_MGMT_REPORTING_ENHANCEMENTS')) {
            this.skip();
        }
    });

    it('should go from left navigation', () => {
        visitVulnerabilityReportingFromLeftNav();
    });

    it('should show a list of report configurations', () => {
        visitVulnerabilityReportingWithFixture('reports/reportConfigurations.json');

        // column headings
        cy.get('th:contains("Report")');
        cy.get('th:contains("Description")');
        cy.get('th:contains("CVE fixability type")');
        cy.get('th:contains("CVE severities")');
        cy.get('th:contains("Last run")');

        // row content
        // name
        cy.get('tbody tr:nth-child(1) td:nth-child(2):contains("Failing report")');
        cy.get('tbody tr:nth-child(2) td:nth-child(2):contains("Successful report")');

        // fixability
        cy.get('tbody tr:nth-child(1) td:nth-child(4):contains("Fixable, Unfixable")');
        cy.get('tbody tr:nth-child(2) td:nth-child(4):contains("Fixable")');

        // severities
        cy.get('tbody tr:nth-child(1) td:nth-child(5):contains("CriticalImportantMediumLow")');
        cy.get('tbody tr:nth-child(2) td:nth-child(5):contains("Critical")');

        // last run
        cy.get('tbody tr:nth-child(1) td:nth-child(6):contains("Error")');
        cy.get('tbody tr:nth-child(2) td:nth-child(6):contains("2022")');
    });
});
