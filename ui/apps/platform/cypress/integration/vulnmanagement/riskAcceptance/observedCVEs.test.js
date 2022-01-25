import * as api from '../../../constants/apiEndpoints';
import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';

const imageEntityPage =
    '/main/vulnerability-management/image/sha256:5469b2315904f5f720034495c3938a4d6f058ec468ce4eca0b1a9291c616c494';

const aliasImageVulnerabilitiesQuery = (req, vulnsQuery, alias) => {
    const { body } = req;
    const matchesQuery = body?.variables?.vulnsQuery === vulnsQuery;
    if (matchesQuery) {
        req.alias = alias;
    }
};

describe('Vulnmanagement Risk Acceptance', () => {
    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_VULN_RISK_MANAGEMENT')) {
            this.skip();
        }
    });

    withAuth();

    describe('Observed CVEs', () => {
        beforeEach(() => {
            cy.intercept('POST', api.riskAcceptance.getImageVulnerabilities, (req) => {
                aliasImageVulnerabilitiesQuery(
                    req,
                    'Vulnerability State:OBSERVED',
                    'getObservedCVEs'
                );
            });
            cy.intercept('POST', api.riskAcceptance.deferVulnerability).as('deferVulnerability');
        });

        it('should be able to defer a CVE', () => {
            cy.visit(imageEntityPage);
            cy.wait('@getObservedCVEs');
            cy.get(
                'table[aria-label="Observed CVEs Table"] tbody tr:first button[aria-label="Actions"]'
            ).click();
            cy.get('li[role="menuitem"] button:contains("Defer CVE")').click();
            cy.get('input[value="2 weeks"]').check();
            cy.get('input[value="All tags within image"]').check();
            cy.get('textarea[id="comment"]').type('Defer for 2 weeks');
            cy.get('button:contains("Request approval")').click();
            cy.wait('@deferVulnerability');
            cy.get(
                'table[aria-label="Observed CVEs Table"] tbody tr:first svg[aria-label="Pending approval icon"]'
            );
        });
    });
});
