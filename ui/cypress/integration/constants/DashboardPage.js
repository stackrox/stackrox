import severityColorMap from '../../../src/constants/severityColors';

export const url = '/main/dashboard';
export const selectors = {
    navLink: 'nav li:contains("Dashboard") a',
    buttons: {
        viewAll: 'button:contains("View All")'
    },
    sectionHeaders: {
        systemViolations: 'h2:contains("System Violations")',
        compliance: 'h2:contains("Compliance")',
        violationsByClusters: 'h2:contains("Violations by Cluster")',
        eventsByTime: 'h2:contains("Active Violations by Time")',
        securityBestPractices: 'h2:contains("Security Best Practices")',
        devopsBestPractices: 'h2:contains("DevOps Best Practices")',
        topRiskyDeployments: 'h2:contains("Top Risky Deployments")'
    },
    chart: {
        xAxis: 'g.xAxis',
        grid: 'g.recharts-cartesian-grid',
        medSeverityBar: `g.recharts-bar-rectangle path[fill="${severityColorMap.MEDIUM_SEVERITY}"]`,
        lowSeverityBar: `g.recharts-bar-rectangle path[fill="${severityColorMap.LOW_SEVERITY}"]`,
        medSeveritySector: `g.recharts-pie-sector path[fill="${severityColorMap.MEDIUM_SEVERITY}"]`,
        legendItem: `span.recharts-legend-item-text`,
        legendLink: '[data-test-id="CIS Docker v1.1.0"]'
    },
    timeseries: 'svg.recharts-surface',
    searchInput: '.react-select__input > input'
};
