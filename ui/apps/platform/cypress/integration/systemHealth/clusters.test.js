import { clustersUrl } from '../../constants/ClustersPage';
import { selectors } from '../../constants/SystemHealth';
import withAuth from '../../helpers/basicAuth';
import { setClock, visitSystemHealth } from '../../helpers/systemHealth';

function visitSystemHealthWithClustersFixtureFilteredByNames(fixturePath, clusterNames) {
    cy.fixture(fixturePath).then(({ clusters }) => {
        visitSystemHealth({
            clusters: {
                body: { clusters: clusters.filter(({ name }) => clusterNames.includes(name)) },
            },
        });
    });
}

describe('System Health Clusters without fixture', () => {
    withAuth();

    it('should go to Clusters via click View All', () => {
        visitSystemHealth();

        cy.get(selectors.clusters.viewAllButton).click();
        cy.location('pathname').should('eq', clustersUrl);
        cy.get('h1:contains("Clusters")');
    });
});

// For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
const currentDatetime = new Date('2020-08-31T13:01:00Z');

const clustersFixturePath = 'clusters/health.json';

describe('System Health Clusters with fixture', () => {
    withAuth();

    const { categoryCount, categoryLabel, healthySubtext, healthyText, problemCount } =
        selectors.clusters;

    it('should have counts in Cluster Overview', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

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
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

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
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

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
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

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
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

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

    const clusterNames = ['eta-7', 'kappa-kilogramme-10', 'lambda-liverpool-11'];

    it('should have counts in Cluster Overview', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

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
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

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
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

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
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

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
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

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

    const clusterNames = ['alpha-amsterdam-1']; // which has Uninitialized status

    it('should have counts in Cluster Overview', () => {
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

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
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

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

    const clusterNames = ['nu-york-13']; // which has Healthy status

    it('should have counts in Cluster Overview', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

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
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

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
