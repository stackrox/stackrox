import withAuth from '../../../helpers/basicAuth';
import {
    interceptAndOverrideFeatureFlags,
    interceptAndOverridePermissions,
} from '../../../helpers/request';
import { visit } from '../../../helpers/visit';
import pf6 from '../../../selectors/pf6';

const vulnerabilityReportsOverviewPath = '/main/vulnerabilities/reports';
const vulnerabilityImageReportsPath = '/main/vulnerabilities/reports/images';
const vulnerabilityNodeReportsPath = '/main/vulnerabilities/reports/nodes';

describe('Vulnerability Reports Overview Navigation', () => {
    withAuth();

    it('navigates to the image vulnerability reports page', () => {
        interceptAndOverridePermissions({ Deployment: 'READ_ACCESS', Image: 'READ_ACCESS' });

        visit(vulnerabilityReportsOverviewPath);

        cy.get(`${pf6.card}:contains("Image vulnerability reports") button`).click();
        cy.location('pathname').should('include', vulnerabilityImageReportsPath);
    });

    it('navigates to the node vulnerability reports page', () => {
        interceptAndOverridePermissions({ Node: 'READ_ACCESS' });
        interceptAndOverrideFeatureFlags({ ROX_NODE_VULNERABILITY_REPORTS: true });

        visit(vulnerabilityReportsOverviewPath);

        cy.get(`${pf6.card}:contains("Node vulnerability reports") button`).click();
        cy.location('pathname').should('include', vulnerabilityNodeReportsPath);
    });
});
