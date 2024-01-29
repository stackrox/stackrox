import withAuth from '../../helpers/basicAuth';

import { visitClusterByNameWithFixture, visitClustersWithFixture } from './Clusters.helpers';

describe('Cluster managedBy', () => {
    withAuth();

    it('should indicate Helm and Operator', () => {
        const fixturePath = 'clusters/health.json';
        visitClustersWithFixture(fixturePath);

        const helmIndicator = '[data-testid="cluster-name"] img[alt="Managed by Helm"]';
        const k8sOperatorIndicator =
            '[data-testid="cluster-name"] img[alt="Managed by a Kubernetes Operator"]';
        const anyIndicator = '[data-testid="cluster-name"] img';
        cy.get(`.rt-tr-group:eq(0) ${helmIndicator}`).should('exist'); // alpha-amsterdam-1
        cy.get(`.rt-tr-group:eq(1) ${anyIndicator}`).should('not.exist'); // epsilon-edison-5
        cy.get(`.rt-tr-group:eq(2) ${k8sOperatorIndicator}`).should('exist'); // eta-7
        cy.get(`.rt-tr-group:eq(3) ${helmIndicator}`).should('exist'); // kappa-kilogramme-10
        cy.get(`.rt-tr-group:eq(4) ${anyIndicator}`).should('not.exist'); // lambda-liverpool-11
        cy.get(`.rt-tr-group:eq(5) ${anyIndicator}`).should('not.exist'); // mu-madegascar-12
        cy.get(`.rt-tr-group:eq(6) ${anyIndicator}`).should('not.exist'); // nu-york-13
    });
});

describe('Cluster configuration', () => {
    withAuth();

    const fixturePath = 'clusters/health.json';

    const assertConfigurationReadOnly = () => {
        [
            'name',
            'mainImage',
            'centralApiEndpoint',
            'collectorImage',
            'admissionControllerEvents',
            'admissionController',
            'admissionControllerUpdates',
            'tolerationsConfig.disabled',
            'slimCollector',
            'dynamicConfig.registryOverride',
            'dynamicConfig.admissionControllerConfig.enabled',
            'dynamicConfig.admissionControllerConfig.enforceOnUpdates',
            'dynamicConfig.admissionControllerConfig.timeoutSeconds',
            'dynamicConfig.admissionControllerConfig.scanInline',
            'dynamicConfig.admissionControllerConfig.disableBypass',
            'dynamicConfig.disableAuditLogs',
        ].forEach((id) => {
            cy.get('[data-testid="cluster-form"]')
                .children()
                .get(`input[id="${id}"]`)
                .should('be.disabled');
        });
        ['Select a cluster type', 'Select a runtime option'].forEach((label) => {
            cy.get('[data-testid="cluster-form"]')
                .children()
                .get(`select[aria-label="${label}"]`)
                .should('be.disabled');
        });
    };

    it('should be read-only for Helm-based installations', () => {
        visitClusterByNameWithFixture('alpha-amsterdam-1', fixturePath);
        assertConfigurationReadOnly();
    });

    it('should be read-only for unknown manager installations that have a defined Helm config', () => {
        visitClusterByNameWithFixture('kappa-kilogramme-10', fixturePath);
        assertConfigurationReadOnly();
    });
});
