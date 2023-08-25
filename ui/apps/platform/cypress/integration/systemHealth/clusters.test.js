import withAuth from '../../helpers/basicAuth';
import { interactAndVisitClusters } from '../clusters/Clusters.helpers';
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

function getCardHeaderDescendantSelector(cardTitle, descendant) {
    return `.pf-c-card__header:has('h2:contains("${cardTitle}")') ${descendant}`;
}

const clustersLink = 'a:contains("View clusters")';

describe('System Health Clusters without fixture', () => {
    withAuth();

    it('should go to Clusters via click link in Cluster status card', () => {
        visitSystemHealth();

        interactAndVisitClusters(() => {
            cy.get(getCardHeaderDescendantSelector('Cluster status', clustersLink)).click();
        });
    });

    it('should go to Clusters via click link in Sensor upgrade card', () => {
        visitSystemHealth();

        interactAndVisitClusters(() => {
            cy.get(getCardHeaderDescendantSelector('Sensor upgrade', clustersLink)).click();
        });
    });

    it('should go to Clusters via click link in Credential expiration card', () => {
        visitSystemHealth();

        interactAndVisitClusters(() => {
            cy.get(getCardHeaderDescendantSelector('Credential expiration', clustersLink)).click();
        });
    });
});

function getCardBodyDescendantSelector(cardTitle, descendant) {
    return `.pf-c-card:has('h2:contains("${cardTitle}")') .pf-c-card__body ${descendant}`;
}

function getTableCellContainsSelector(cardTitle, nthRow, dataLabel, contents) {
    return getCardBodyDescendantSelector(
        cardTitle,
        `tbody tr:nth-child(${nthRow}) td[data-label="${dataLabel}"]:contains("${contents}")`
    );
}

// For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
const currentDatetime = new Date('2020-08-31T13:01:00Z');

const clustersFixturePath = 'clusters/health.json';

describe('System Health Clusters with fixture', () => {
    withAuth();

    it('should have phrase in Cluster status card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

        const cardTitle = 'Cluster status';

        cy.get(getCardHeaderDescendantSelector(cardTitle, 'div:contains("2 unhealthy")'));
    });

    it('should have counts in row 1 of Cluster status card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

        const cardTitle = 'Cluster status';
        const nthRow = 1; // Clusters

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 2));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 2));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 2));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 7));
    });

    it('should have counts in row 2 of Cluster status card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

        const cardTitle = 'Cluster status';
        const nthRow = 2; // Clusters because of sensor

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 4));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 7));
    });

    it('should have counts in row 3 of Cluster status', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

        const cardTitle = 'Cluster status';
        const nthRow = 3; // Clusters because of collector

        // Unavailable from fixture has inconsistent value from first 2 tests.
        // Okay for testing, but unlikely to occur in reality.
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 3));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 7));
    });

    it('should have counts in row 4 of Cluster status', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

        const cardTitle = 'Cluster status';
        const nthRow = 4; // Clusters because of admission control

        // Unavailable from fixture has inconsistent value from first 2 tests.
        // Okay for testing, but unlikely to occur in reality.
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 4));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 7));
    });

    it('should have phrase in Sensor upgrade card header', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

        const cardTitle = 'Sensor upgrade';

        cy.get(getCardHeaderDescendantSelector(cardTitle, 'div:contains("2 degraded")'));
    });

    it('should have counts in Sensor upgrade card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

        const cardTitle = 'Sensor upgrade';
        const nthRow = 1; // Clusters

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Up to date', 4));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Failed', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Out of date', 2));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 7));
    });

    it('should have phrase in Credential expiration card header', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

        const cardTitle = 'Credential expiration';

        cy.get(getCardHeaderDescendantSelector(cardTitle, 'div:contains("1 unhealthy")'));
    });

    it('should have counts in Credential expiration card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealth({
            clusters: { fixture: clustersFixturePath },
        });

        const cardTitle = 'Credential expiration';
        const nthRow = 1; // Clusters

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, '\u2265 30 days', 4));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, '< 7 days', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, '< 30 days', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 7));
    });
});

describe('System Health Clusters subset 3', () => {
    withAuth();

    const clusterNames = ['eta-7', 'kappa-kilogramme-10', 'lambda-liverpool-11'];

    it('should have phrase in Cluster status card header', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Cluster status';

        cy.get(getCardHeaderDescendantSelector(cardTitle, 'div:contains("1 unhealthy")'));
    });

    it('should have counts in row 1 of Cluster status card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Cluster status';
        const nthRow = 1; // Clusters

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 2));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 3));
    });

    it('should have counts in row 2 of Cluster status card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Cluster status';
        const nthRow = 2; // Clusters because of sensor

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 2));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 3));
    });

    it('should have counts in row 3 of Cluster status card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Cluster status';
        const nthRow = 3; // Clusters because of collector

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 3));
    });

    it('should have counts in row 4 of Cluster status card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Cluster status';
        const nthRow = 4; // Clusters because of admission control

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 2));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 3));
    });

    it('should have not have counts in Sensor upgrade card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Sensor upgrade';

        cy.get(getCardHeaderDescendantSelector(cardTitle, 'div:contains("3 healthy")'));
        cy.get(getCardBodyDescendantSelector(cardTitle, 'table')).should('not.exist');
    });

    it('should have phrase in Credential expiration card header', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Credential expiration';

        cy.get(getCardHeaderDescendantSelector(cardTitle, 'div:contains("1 degraded")'));
    });

    it('should have counts in Credential expiration card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Credential expiration';
        const nthRow = 1; // Clusters

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, '\u2265 30 days', 2));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, '< 7 days', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, '< 30 days', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 3));
    });
});

describe('System Health Clusters subset 1 Uninitialized', () => {
    withAuth();

    const clusterNames = ['alpha-amsterdam-1']; // which has Uninitialized status

    // No phrases in card header if no unhealthy, degraded, nor healthy.

    it('should have counts in row 1 of Cluster status card', () => {
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Cluster status';
        const nthRow = 1; // Clusters

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 1));
    });

    it('should have counts in row 2 of Cluster status card', () => {
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Cluster status';
        const nthRow = 2; // Clusters because of sensor

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 1));
    });

    it('should have counts in row 3 of Cluster status card', () => {
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Cluster status';
        const nthRow = 3; // Clusters because of collector

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 1));
    });

    it('should have counts in row 4 of Cluster status card', () => {
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Cluster status';
        const nthRow = 4; // Clusters because of admission control

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Healthy', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unhealthy', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Degraded', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 1));
    });

    it('should have counts in Sensor upgrade card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Sensor upgrade';
        const nthRow = 1; // Clusters

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Up to date', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Failed', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Out of date', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 1));
    });

    it('should have counts in Credential expiration card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Credential expiration';
        const nthRow = 1; // Clusters

        cy.get(getTableCellContainsSelector(cardTitle, nthRow, '\u2265 30 days', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, '< 7 days', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, '< 30 days', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Unavailable', 0));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Uninitialized', 1));
        cy.get(getTableCellContainsSelector(cardTitle, nthRow, 'Total', 1));
    });
});

describe('System Health Clusters subset 1 Healthy', () => {
    withAuth();

    const clusterNames = ['nu-york-13']; // which has Healthy status

    it('should have not have counts in Cluster status card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Cluster status';

        cy.get(getCardHeaderDescendantSelector(cardTitle, 'div:contains("1 healthy")'));
        cy.get(getCardBodyDescendantSelector(cardTitle, 'table')).should('not.exist');
    });

    it('should have not have counts in Sensor upgrade card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Sensor upgrade';

        cy.get(getCardHeaderDescendantSelector(cardTitle, 'div:contains("1 healthy")'));
        cy.get(getCardBodyDescendantSelector(cardTitle, 'table')).should('not.exist');
    });

    it('should have not have counts in Credential expiration card', () => {
        setClock(currentDatetime); // call before visit
        visitSystemHealthWithClustersFixtureFilteredByNames(clustersFixturePath, clusterNames);

        const cardTitle = 'Credential expiration';

        cy.get(getCardHeaderDescendantSelector(cardTitle, 'div:contains("1 healthy")'));
        cy.get(getCardBodyDescendantSelector(cardTitle, 'table')).should('not.exist');
    });
});
