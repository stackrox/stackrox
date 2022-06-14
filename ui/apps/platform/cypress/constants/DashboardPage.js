import { severityColorMap } from '../../src/constants/severityColors';
import scopeSelectors from '../helpers/scopeSelectors';
import navigationSelectors from '../selectors/navigation';

// TODO Make `pfUrl` the default url once phase one of the PF Dashboard is enabled
export const url = '/main/dashboard';
export const pfUrl = '/main/dashboard-pf';

export const selectors = {
    navLink: `${navigationSelectors.navLinks}:contains("Dashboard")`,
    buttons: {
        viewAll: 'button:contains("View All")',
    },
    summaryCount: '[data-testid="summary-tile-count"]',
    sectionHeaders: {
        systemViolations: 'h2:contains("System Violations")',
        compliance: 'h2:contains("Compliance")',
        eventsByTime: 'h2:contains("Active Violations by Time")',
        securityBestPractices: 'h2:contains("Security Best Practices")',
        devopsBestPractices: 'h2:contains("DevOps Best Practices")',
        topRiskyDeployments: 'h2:contains("Top Risky Deployments")',
    },
    chart: scopeSelectors('h2:contains("Violations by Cluster") + div', {
        xAxis: 'g.xAxis',
        grid: 'g.recharts-cartesian-grid',
        medSeverityBar: `g.recharts-bar-rectangle path[fill="${severityColorMap.MEDIUM_SEVERITY}"]`,
        lowSeverityBar: `g.recharts-bar-rectangle path[fill="${severityColorMap.LOW_SEVERITY}"]`,
        medSeveritySector: `g.recharts-pie-sector path[fill="${severityColorMap.MEDIUM_SEVERITY}"]`,
        legendItem: `span.recharts-legend-item-text`,
        legendLink: '[data-testid="CIS Docker v1.2.0"]',
        resultsMessage: '[data-testid="results-message"]',
    }),
    timeseries: 'svg.recharts-surface',
    searchInput: '.react-select__input > input',
    severityTile: '[data-testid="severity-tile"]',
    topRiskyDeployments: '[data-testid="top-risky-deployments"] ul li a',
    policyCategoryViolations: '[data-testid="policy-category-violation"]',
};
