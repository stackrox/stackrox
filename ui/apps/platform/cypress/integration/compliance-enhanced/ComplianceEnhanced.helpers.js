import { visit } from '../../helpers/visit';

// path

export const basePath = '/main/compliance-enhanced';

// visit

export function visitComplianceEnhancedDashboard() {
    // TODO: (vjw, 1 Nov 2023) add routes matchers to this function, after API is available for Status
    visit(`${basePath}/status`);

    cy.get(`h1:contains("Compliance")`);
}
