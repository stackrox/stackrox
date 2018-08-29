export const url = '/main/dashboard';

const severityColors = {
    CRITICAL_SEVERITY: 'hsl(7, 100%, 55%)',
    HIGH_SEVERITY: 'hsl(349, 100%, 78%)',
    MEDIUM_SEVERITY: 'hsl(20, 100%, 78%)',
    LOW_SEVERITY: 'hsl(42, 100%, 84%)'
};

export const selectors = {
    navLink: 'nav li:contains("Dashboard") a',
    buttons: {
        more: 'button:contains("More")'
    },
    sectionHeaders: {
        environmentRisk: 'h2:contains("Environment Risk")',
        benchmarks: 'h2:contains("Benchmarks")',
        violationsByClusters: 'h2:contains("Violations by Cluster")',
        eventsByTime: 'h2:contains("Active Violations by Time")',
        securityBestPractices: 'h2:contains("Security Best Practices")',
        devopsBestPractices: 'h2:contains("DevOps Best Practices")',
        topRiskyDeployments: 'h2:contains("Top Risky Deployments")'
    },
    chart: {
        xAxis: 'g.xAxis',
        grid: 'g.recharts-cartesian-grid',
        medSeverityBar: `g.recharts-bar-rectangle path[fill="${severityColors.MEDIUM_SEVERITY}"]`,
        lowSeverityBar: `g.recharts-bar-rectangle path[fill="${severityColors.LOW_SEVERITY}"]`,
        medSeveritySector: `g.recharts-pie-sector path[fill="${severityColors.MEDIUM_SEVERITY}"]`,
        legendItem: `span.recharts-legend-item-text`
    },
    slick: {
        dashboardBenchmarks: {
            prevButton: '.dashboard-benchmarks .carousel-prev-arrow',
            nextButton: '.dashboard-benchmarks .carousel-next-arrow',
            list: '.dashboard-benchmarks .slick-slide',
            currentSlide: '.dashboard-benchmarks .slick-current',
            track: '.slick-track'
        }
    },
    timeseries: 'svg.recharts-surface',
    searchInput: '.Select-input > input'
};
