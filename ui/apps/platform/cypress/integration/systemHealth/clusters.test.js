import { selectors, systemHealthUrl } from '../../constants/SystemHealth';
import { clusters as clustersApi } from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import navigationSelectors from '../../selectors/navigation';

describe('System Health Clusters local deployment', () => {
    withAuth();

    beforeEach(() => {
        cy.intercept('GET', clustersApi.list).as('GetClusters');
    });

    it('should go from left navigation to Dashboard and have widgets', () => {
        cy.visit('/');
        cy.get(`${navigationSelectors.navExpandable}:contains("Platform Configuration")`).click();
        cy.get(`${navigationSelectors.nestedNavLinks}:contains("System Health")`).click();
        cy.wait('@GetClusters');

        cy.get('[data-testid="header-text"]').should('have.text', 'System Health');

        Object.entries({
            clusterOverview: 'Cluster Overview',
            collectorStatus: 'Collector Status',
            sensorStatus: 'Sensor Status',
            sensorUpgrade: 'Sensor Upgrade',
            credentialExpiration: 'Credential Expiration',
        }).forEach(([key, text]) => {
            cy.get(`${selectors.clusters.widgets[key]} [data-testid="widget-header"]`).should(
                'have.text',
                text
            );
        });
    });

    it('should go from Dashboard to Clusters via click View All', () => {
        cy.visit(systemHealthUrl);
        cy.wait('@GetClusters');

        cy.get(selectors.clusters.viewAllButton).click();
        cy.wait('@GetClusters');

        cy.get('[data-testid="header-text"]').should('have.text', 'Clusters');
        cy.get('[data-testid="clusters-side-panel-header"]').should('not.exist');
    });
});

describe('System Health Clusters health fixture', () => {
    withAuth();

    const { categoryCount, categoryLabel, healthySubtext, healthyText, problemCount } =
        selectors.clusters;

    // For comparison to `sensorCertExpiry` in clusters fixture.
    const currentDatetime = new Date('2020-08-31T13:01:00Z');

    beforeEach(() => {
        cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

        cy.intercept('GET', clustersApi.list, {
            fixture: 'clusters/health.json',
        }).as('GetClusters');

        cy.visit(systemHealthUrl);
        cy.wait('@GetClusters');
    });

    it('should have counts in Cluster Overview', () => {
        const widgetSelector = selectors.clusters.widgets.clusterOverview;

        Object.entries({
            HEALTHY: {
                label: 'Healthy',
                count: 2,
            },
            UNINITIALIZED: {
                label: 'Uninitialized',
                count: 1,
            },
            DEGRADED: {
                label: 'Degraded',
                count: 2,
            },
            UNHEALTHY: {
                label: 'Unhealthy',
                count: 2,
            },
        }).forEach(([key, { label, count }]) => {
            const itemSelector = `${widgetSelector} [data-testid="${key}"]`;
            cy.get(`${itemSelector} ${categoryLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${categoryCount}`).should('have.text', String(count));
        });
    });

    it('should have counts in Collector Status', () => {
        const widgetSelector = selectors.clusters.widgets.collectorStatus;
        let total = 0;

        Object.entries({
            DEGRADED: {
                label: 'Degraded',
                count: 1,
            },
            UNHEALTHY: {
                label: 'Unhealthy',
                count: 1,
            },
        }).forEach(([key, { label, count }]) => {
            const itemSelector = `${widgetSelector} [data-testid="${key}"]`;
            cy.get(`${itemSelector} ${categoryLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${categoryCount}`).should('have.text', String(count));
            total += count;
        });

        cy.get(`${widgetSelector} ${healthyText}`).should('not.exist');
        cy.get(`${widgetSelector} ${healthySubtext}`).should('not.exist');
        cy.get(`${widgetSelector} ${problemCount}`).should('have.text', String(total));
    });

    it('should have counts in Sensor Status', () => {
        const widgetSelector = selectors.clusters.widgets.sensorStatus;
        let total = 0;

        Object.entries({
            DEGRADED: {
                label: 'Degraded',
                count: 1,
            },
            UNHEALTHY: {
                label: 'Unhealthy',
                count: 1,
            },
        }).forEach(([key, { label, count }]) => {
            const itemSelector = `${widgetSelector} [data-testid="${key}"]`;
            cy.get(`${itemSelector} ${categoryLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${categoryCount}`).should('have.text', String(count));
            total += count;
        });

        cy.get(`${widgetSelector} ${healthyText}`).should('not.exist');
        cy.get(`${widgetSelector} ${healthySubtext}`).should('not.exist');
        cy.get(`${widgetSelector} ${problemCount}`).should('have.text', String(total));
    });

    it('should have counts in Sensor Updgrade', () => {
        const widgetSelector = selectors.clusters.widgets.sensorUpgrade;
        let total = 0;

        Object.entries({
            download: {
                label: 'Upgrade available',
                count: 2,
            },
        }).forEach(([key, { label, count }]) => {
            const itemSelector = `${widgetSelector} [data-testid="${key}"]`;
            cy.get(`${itemSelector} ${categoryLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${categoryCount}`).should('have.text', String(count));
            total += count;
        });

        cy.get(`${widgetSelector} ${healthyText}`).should('not.exist');
        cy.get(`${widgetSelector} ${healthySubtext}`).should('not.exist');
        cy.get(`${widgetSelector} ${problemCount}`).should('have.text', String(total));
    });

    it('should have counts in Credential Expiration', () => {
        const widgetSelector = selectors.clusters.widgets.credentialExpiration;
        let total = 0;

        Object.entries({
            DEGRADED: {
                label: 'Expiring in < 30 days',
                count: 1,
            },
            UNHEALTHY: {
                label: 'Expiring in < 7 days',
                count: 1,
            },
        }).forEach(([key, { label, count }]) => {
            const itemSelector = `${widgetSelector} [data-testid="${key}"]`;
            cy.get(`${itemSelector} ${categoryLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${categoryCount}`).should('have.text', String(count));
            total += count;
        });

        cy.get(`${widgetSelector} ${healthyText}`).should('not.exist');
        cy.get(`${widgetSelector} ${healthySubtext}`).should('not.exist');
        cy.get(`${widgetSelector} ${problemCount}`).should('have.text', String(total));
    });
});

describe('System Health Clusters subset 3', () => {
    withAuth();

    const { categoryCount, categoryLabel, healthySubtext, healthyText, problemCount } =
        selectors.clusters;

    // For comparison to `sensorCertExpiry` in clusters fixture.
    const currentDatetime = new Date('2020-08-31T13:01:00Z');

    let clusters;

    before(() => {
        cy.fixture('clusters/health.json').then((response) => {
            // Return eta-7, kappa-kilogramme-10, lambda-liverpool-11.
            clusters = response.clusters.slice(2, 5);
        });
    });

    beforeEach(() => {
        cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

        cy.intercept('GET', clustersApi.list, {
            body: { clusters },
        }).as('GetClusters');

        cy.visit(systemHealthUrl);
        cy.wait('@GetClusters');
    });

    it('should have counts in Cluster Overview', () => {
        const widgetSelector = selectors.clusters.widgets.clusterOverview;

        Object.entries({
            HEALTHY: {
                label: 'Healthy',
                count: 0,
            },
            UNINITIALIZED: {
                label: 'Uninitialized',
                count: 0,
            },
            DEGRADED: {
                label: 'Degraded',
                count: 2,
            },
            UNHEALTHY: {
                label: 'Unhealthy',
                count: 1,
            },
        }).forEach(([key, { label, count }]) => {
            const itemSelector = `${widgetSelector} [data-testid="${key}"]`;
            cy.get(`${itemSelector} ${categoryLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${categoryCount}`).should('have.text', String(count));
        });
    });

    it('should have problem counts in Collector Status', () => {
        const widgetSelector = selectors.clusters.widgets.collectorStatus;
        let total = 0;

        Object.entries({
            DEGRADED: {
                label: 'Degraded',
                count: 1,
            },
            UNHEALTHY: {
                label: 'Unhealthy',
                count: 1,
            },
        }).forEach(([key, { label, count }]) => {
            const itemSelector = `${widgetSelector} [data-testid="${key}"]`;
            cy.get(`${itemSelector} ${categoryLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${categoryCount}`).should('have.text', String(count));
            total += count;
        });

        cy.get(`${widgetSelector} ${healthyText}`).should('not.exist');
        cy.get(`${widgetSelector} ${healthySubtext}`).should('not.exist');
        cy.get(`${widgetSelector} ${problemCount}`).should('have.text', String(total));
    });

    it('should have problem count in Sensor Status', () => {
        const widgetSelector = selectors.clusters.widgets.sensorStatus;
        let total = 0;

        Object.entries({
            DEGRADED: {
                label: 'Degraded',
                count: 1,
            },
        }).forEach(([key, { label, count }]) => {
            const itemSelector = `${widgetSelector} [data-testid="${key}"]`;
            cy.get(`${itemSelector} ${categoryLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${categoryCount}`).should('have.text', String(count));
            total += count;
        });

        cy.get(`${widgetSelector} ${healthyText}`).should('not.exist');
        cy.get(`${widgetSelector} ${healthySubtext}`).should('not.exist');
        cy.get(`${widgetSelector} ${problemCount}`).should('have.text', String(total));
    });

    it('should have healthy count in Sensor Updgrade', () => {
        const widgetSelector = selectors.clusters.widgets.sensorUpgrade;
        const nbsp = '\u00A0';

        cy.get(`${widgetSelector} ${healthyText}`).should(
            'have.text',
            `3 clusters up${nbsp}to${nbsp}date with${nbsp}central`
        );
        cy.get(`${widgetSelector} ${healthySubtext}`).should(
            'have.text',
            'All sensor versions match central version'
        );
        cy.get(`${widgetSelector} ${categoryLabel}`).should('not.exist');
        cy.get(`${widgetSelector} ${categoryCount}`).should('not.exist');
        cy.get(`${widgetSelector} ${problemCount}`).should('not.exist');
    });

    it('should have problem count in Credential Expiration', () => {
        const widgetSelector = selectors.clusters.widgets.credentialExpiration;
        let total = 0;

        Object.entries({
            DEGRADED: {
                label: 'Expiring in < 30 days',
                count: 1,
            },
        }).forEach(([key, { label, count }]) => {
            const itemSelector = `${widgetSelector} [data-testid="${key}"]`;
            cy.get(`${itemSelector} ${categoryLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${categoryCount}`).should('have.text', String(count));
            total += count;
        });

        cy.get(`${widgetSelector} ${healthyText}`).should('not.exist');
        cy.get(`${widgetSelector} ${healthySubtext}`).should('not.exist');
        cy.get(`${widgetSelector} ${problemCount}`).should('have.text', String(total));
    });
});

describe('System Health Clusters subset 1 Uninitialized', () => {
    withAuth();

    const { categoryCount, categoryLabel, healthySubtext, healthyText, problemCount } =
        selectors.clusters;

    let clusters;

    before(() => {
        cy.fixture('clusters/health.json').then((response) => {
            // Return only alpha-amsterdam-1, which has Uninitialized status.
            clusters = response.clusters.slice(0, 1);
        });
    });

    beforeEach(() => {
        cy.intercept('GET', clustersApi.list, {
            body: { clusters },
        }).as('GetClusters');

        cy.visit(systemHealthUrl);
        cy.wait('@GetClusters');
    });

    it('should have counts in Cluster Overview', () => {
        const widgetSelector = selectors.clusters.widgets.clusterOverview;

        Object.entries({
            HEALTHY: {
                label: 'Healthy',
                count: 0,
            },
            UNINITIALIZED: {
                label: 'Uninitialized',
                count: 1,
            },
            DEGRADED: {
                label: 'Degraded',
                count: 0,
            },
            UNHEALTHY: {
                label: 'Unhealthy',
                count: 0,
            },
        }).forEach(([key, { label, count }]) => {
            const itemSelector = `${widgetSelector} [data-testid="${key}"]`;
            cy.get(`${itemSelector} ${categoryLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${categoryCount}`).should('have.text', String(count));
        });
    });

    it('should have 0 clusters in other widgets', () => {
        const { collectorStatus, credentialExpiration, sensorStatus, sensorUpgrade } =
            selectors.clusters.widgets;
        [collectorStatus, sensorStatus, sensorUpgrade, credentialExpiration].forEach(
            (widgetSelector) => {
                cy.get(`${widgetSelector} ${healthyText}`).should('contain', '0 clusters');
                cy.get(`${widgetSelector} ${healthySubtext}`).should('not.exist');
                cy.get(`${widgetSelector} ${categoryLabel}`).should('not.exist');
                cy.get(`${widgetSelector} ${categoryCount}`).should('not.exist');
                cy.get(`${widgetSelector} ${problemCount}`).should('not.exist');
            }
        );
    });
});

describe('System Health Clusters subset 1 Healthy', () => {
    withAuth();

    const { categoryCount, categoryLabel, healthySubtext, healthyText, problemCount } =
        selectors.clusters;

    // For comparison to `sensorCertExpiry` in clusters fixture.
    const currentDatetime = new Date('2020-08-31T13:01:00Z');

    let clusters;

    before(() => {
        cy.fixture('clusters/health.json').then((response) => {
            // Return only nu-york-13, which has Healthy status.
            clusters = response.clusters.slice(6, 7);
        });
    });

    beforeEach(() => {
        cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

        cy.intercept('GET', clustersApi.list, {
            body: { clusters },
        }).as('GetClusters');

        cy.visit(systemHealthUrl);
        cy.wait('@GetClusters');
    });

    it('should have counts in Cluster Overview', () => {
        const widgetSelector = selectors.clusters.widgets.clusterOverview;

        Object.entries({
            HEALTHY: {
                label: 'Healthy',
                count: 1,
            },
            UNINITIALIZED: {
                label: 'Uninitialized',
                count: 0,
            },
            DEGRADED: {
                label: 'Degraded',
                count: 0,
            },
            UNHEALTHY: {
                label: 'Unhealthy',
                count: 0,
            },
        }).forEach(([key, { label, count }]) => {
            const itemSelector = `${widgetSelector} [data-testid="${key}"]`;
            cy.get(`${itemSelector} ${categoryLabel}`).should('have.text', label);
            cy.get(`${itemSelector} ${categoryCount}`).should('have.text', String(count));
        });
    });

    it('should have 1 cluster in other widgets', () => {
        Object.entries({
            collectorStatus: 'All expected collector pods are ready',
            sensorStatus: 'All sensors last contacted less than 1 minute ago',
            sensorUpgrade: 'All sensor versions match central version',
            credentialExpiration: 'There are no credential expirations this month',
        }).forEach(([key, subtext]) => {
            const widgetSelector = selectors.clusters.widgets[key];
            cy.get(`${widgetSelector} ${healthyText}`).should('contain', '1 cluster');
            cy.get(`${widgetSelector} ${healthySubtext}`).should('have.text', subtext);
            cy.get(`${widgetSelector} ${categoryLabel}`).should('not.exist');
            cy.get(`${widgetSelector} ${categoryCount}`).should('not.exist');
            cy.get(`${widgetSelector} ${problemCount}`).should('not.exist');
        });
    });
});

describe('System Health, PatternFly version', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_SYSTEM_HEALTH_PF')) {
            this.skip();
        }
    });

    beforeEach(() => {
        cy.intercept('GET', clustersApi.list).as('GetClusters');
    });

    it('should go from left navigation to Dashboard and have widgets', () => {
        cy.visit('/');
        cy.get(`${navigationSelectors.navExpandable}:contains("Platform Configuration")`).click();
        cy.get(`${navigationSelectors.nestedNavLinks}:contains("System Health")`).click();

        // TODO: remove this direct access shim after the PF version of the page is the default
        cy.visit('/main/system-health-pf');

        cy.wait('@GetClusters');

        cy.get('h1:contains("System Health")');
    });
});
