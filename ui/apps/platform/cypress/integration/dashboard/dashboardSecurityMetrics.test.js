import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { visitMainDashboardPF } from '../../helpers/main';

import { pfSelectors as selectors } from '../../constants/DashboardPage';

function makeFixtureCounts(crit, high, med, low) {
    return [
        { severity: 'CRITICAL_SEVERITY', count: `${crit}` },
        { severity: 'HIGH_SEVERITY', count: `${high}` },
        { severity: 'MEDIUM_SEVERITY', count: `${med}` },
        { severity: 'LOW_SEVERITY', count: `${low}` },
    ];
}

const policyViolationsByCategory = {
    groups: [
        { counts: makeFixtureCounts(5, 20, 30, 10), group: 'Anomalous Activity' },
        { counts: makeFixtureCounts(5, 2, 30, 5), group: 'Docker CIS' },
        { counts: makeFixtureCounts(10, 20, 5, 5), group: 'Network Tools' },
        { counts: makeFixtureCounts(15, 2, 10, 5), group: 'Security Best Practices' },
        { counts: makeFixtureCounts(20, 10, 2, 10), group: 'Privileges' },
        { counts: makeFixtureCounts(15, 8, 10, 5), group: 'Vulnerability Management' },
    ],
};

describe('Dashboard security metrics phase one action widgets', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_SECURITY_METRICS_PHASE_ONE')) {
            this.skip();
        }
    });

    it('should visit patternfly dashboard', () => {
        visitMainDashboardPF();

        cy.get(selectors.pageHeader);
    });

    it('should sort a policy violations by category widget by severity and volume of violations', () => {
        visitMainDashboardPF();

        cy.intercept('GET', api.alerts.countsByCategory, {
            body: policyViolationsByCategory,
        }).as('getPolicyViolationsByCategory');
        cy.wait('@getPolicyViolationsByCategory');

        const widgetSelectors = selectors.violationsByCategory;

        // Default sorting should be by severity of critical and high Violations, with critical taking priority.
        cy.get(`${widgetSelectors.axisLabel(0, 4)}:contains('Privileges')`);
        cy.get(`${widgetSelectors.axisLabel(0, 3)}:contains('Vulnerability Management')`);
        cy.get(`${widgetSelectors.axisLabel(0, 2)}:contains('Security Best Practices')`);
        cy.get(`${widgetSelectors.axisLabel(0, 1)}:contains('Network Tools')`);
        cy.get(`${widgetSelectors.axisLabel(0, 0)}:contains('Anomalous Activity')`);

        // Switch to sort-by-volume, which orders the chart by total violations per category
        cy.get(widgetSelectors.optionsToggle).click();
        cy.get(widgetSelectors.volumeOption).click();
        cy.get(widgetSelectors.optionsToggle).click();

        cy.get(`${widgetSelectors.axisLabel(0, 4)}:contains('Network Tools')`);
        cy.get(`${widgetSelectors.axisLabel(0, 3)}:contains('Privileges')`);
        cy.get(`${widgetSelectors.axisLabel(0, 2)}:contains('Anomalous Activity')`);
        cy.get(`${widgetSelectors.axisLabel(0, 1)}:contains('Vulnerability Management')`);
        cy.get(`${widgetSelectors.axisLabel(0, 0)}:contains('Security Best Practices')`);
    });

    it('should allow toggling of severities for a policy violations by category widget', () => {
        visitMainDashboardPF();

        cy.intercept('GET', api.alerts.countsByCategory, {
            body: policyViolationsByCategory,
        }).as('getPolicyViolationsByCategory');
        cy.wait('@getPolicyViolationsByCategory');

        const widgetSelectors = selectors.violationsByCategory;

        // Sort by volume, so that enabling lower severity bars changes the order of the chart
        cy.get(widgetSelectors.optionsToggle).click();
        cy.get(widgetSelectors.volumeOption).click();
        cy.get(widgetSelectors.optionsToggle).click();

        // Toggle on low and medium violations, which are disabled by default
        cy.get(widgetSelectors.legendLabel(2)).click();
        cy.get(widgetSelectors.legendLabel(3)).click();

        cy.get(`${widgetSelectors.axisLabel(0, 4)}:contains('Anomalous Activity')`);
        cy.get(`${widgetSelectors.axisLabel(0, 3)}:contains('Docker CIS')`);
        cy.get(`${widgetSelectors.axisLabel(0, 2)}:contains('Privileges')`);
        cy.get(`${widgetSelectors.axisLabel(0, 1)}:contains('Network Tools')`);
        cy.get(`${widgetSelectors.axisLabel(0, 0)}:contains('Vulnerability Management')`);
    });
});
