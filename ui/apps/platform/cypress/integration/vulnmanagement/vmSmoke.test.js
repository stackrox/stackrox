import { url } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { visitVulnerabilityManagementDashboardFromLeftNav } from '../../helpers/vulnmanagement/entities';

describe('Smoke test for vulnmanagement', () => {
    withAuth();

    it('VulnManagement tile link is present and lands on dashboard page', () => {
        visitVulnerabilityManagementDashboardFromLeftNav();
        cy.location('pathname').should('eq', url.dashboard);
    });
});
