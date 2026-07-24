import { mount } from '@cypress/react18';

import SensorCompatibilityPanel from './SensorCompatibilityPanel';
import type { SensorVersionCompatibility } from 'types/cluster.proto';

describe('SensorCompatibilityPanel', () => {
    const compatibleVersions = ['4.10.0', '4.11.0', '4.12.0'];
    const sensorVersion = '4.10.0';
    const centralVersion = '4.11.0';

    it('should render panel with Matched status and no guidance', () => {
        const compatibility: SensorVersionCompatibility = 'MATCHED';
        mount(
            <SensorCompatibilityPanel
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
                sensorVersion={sensorVersion}
                centralVersion={centralVersion}
            />
        );

        cy.findByText('Sensor compatibility status').should('exist');
        cy.findByTestId('sensorCompatibility').should('contain', 'Matched');
        cy.findByText('Guidance').should('not.exist');
    });

    it('should render panel with Compatible status and guidance', () => {
        const compatibility: SensorVersionCompatibility = 'COMPATIBLE_BEHIND';
        mount(
            <SensorCompatibilityPanel
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
                sensorVersion={sensorVersion}
                centralVersion={centralVersion}
            />
        );

        cy.findByTestId('sensorCompatibility').should('contain', 'Compatible (Behind)');
        cy.findByText('Guidance').should('exist');
        cy.contains(
            'The Sensor version is older than Central but still compatible. Consider upgrading Sensor to match Central.'
        ).should('exist');
    });

    it('should render panel with Incompatible status and guidance (non-Helm)', () => {
        const compatibility: SensorVersionCompatibility = 'INCOMPATIBLE_BEHIND';
        mount(
            <SensorCompatibilityPanel
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
                sensorVersion={sensorVersion}
                centralVersion={centralVersion}
                managedBy="MANAGER_TYPE_KUBERNETES_OPERATOR"
            />
        );

        cy.findByTestId('sensorCompatibility').should('contain', 'Incompatible (Behind)');
        cy.findByText('Guidance').should('exist');
        cy.contains(
            'The Sensor version is too old and incompatible with Central. Upgrade Sensor to a supported version.'
        ).should('exist');
        cy.contains('Since this cluster is Helm-managed').should('not.exist');
    });

    it('should render panel with Incompatible (Ahead) status and guidance to upgrade Central', () => {
        const compatibility: SensorVersionCompatibility = 'INCOMPATIBLE_AHEAD';
        mount(
            <SensorCompatibilityPanel
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
                sensorVersion={sensorVersion}
                centralVersion={centralVersion}
                managedBy="MANAGER_TYPE_HELM_CHART"
            />
        );

        cy.findByTestId('sensorCompatibility').should('contain', 'Incompatible (Ahead)');
        cy.findByText('Guidance').should('exist');
        cy.contains(
            'The Sensor version is too new and incompatible with Central. Upgrade Central to a supported version.'
        ).should('exist');
        cy.contains('Since this cluster is Helm-managed').should('not.exist');
    });

    it('should render panel with Incompatible (Behind) status and Helm guidance', () => {
        const compatibility: SensorVersionCompatibility = 'INCOMPATIBLE_BEHIND';
        mount(
            <SensorCompatibilityPanel
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
                sensorVersion={sensorVersion}
                centralVersion={centralVersion}
                managedBy="MANAGER_TYPE_HELM_CHART"
            />
        );

        cy.findByTestId('sensorCompatibility').should('contain', 'Incompatible (Behind)');
        cy.findByText('Guidance').should('exist');
        cy.contains('Since this cluster is Helm-managed, upgrade using Helm.').should('exist');
    });

    it('should render sensor and central versions', () => {
        const compatibility: SensorVersionCompatibility = 'MATCHED';
        mount(
            <SensorCompatibilityPanel
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
                sensorVersion={sensorVersion}
                centralVersion={centralVersion}
            />
        );

        cy.findByText('Sensor version').should('exist');
        cy.contains(sensorVersion).should('exist');
        cy.findByText('Central version').should('exist');
        cy.contains(centralVersion).should('exist');
    });

    it('should render supported sensor versions list', () => {
        const compatibility: SensorVersionCompatibility = 'MATCHED';
        mount(
            <SensorCompatibilityPanel
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
                sensorVersion={sensorVersion}
                centralVersion={centralVersion}
            />
        );

        cy.findByText('Supported sensor versions').should('exist');
        compatibleVersions.forEach((version) => {
            cy.contains(version).should('exist');
        });
    });

    it('should render "Not available" when compatible versions list is empty', () => {
        const compatibility: SensorVersionCompatibility = 'MATCHED';
        mount(
            <SensorCompatibilityPanel
                compatibility={compatibility}
                compatibleVersions={[]}
                sensorVersion={sensorVersion}
                centralVersion={centralVersion}
            />
        );

        cy.findByText('Supported sensor versions').should('exist');
        cy.contains('Not available').should('exist');
    });

    it('should render documentation link', () => {
        const compatibility: SensorVersionCompatibility = 'COMPATIBLE_BEHIND';
        mount(
            <SensorCompatibilityPanel
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
                sensorVersion={sensorVersion}
                centralVersion={centralVersion}
            />
        );

        cy.findByText('See documentation')
            .should('exist')
            .should('have.attr', 'href', 'https://access.redhat.com/articles/7045053')
            .should('have.attr', 'target', '_blank');
    });

    it('should render tooltip for supported sensor versions', () => {
        const compatibility: SensorVersionCompatibility = 'MATCHED';
        mount(
            <SensorCompatibilityPanel
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
                sensorVersion={sensorVersion}
                centralVersion={centralVersion}
            />
        );

        cy.findByText('Supported sensor versions').should('exist');
        cy.get('[data-ouia-component-type="PF6/Tooltip"]').should('exist');
    });
});
