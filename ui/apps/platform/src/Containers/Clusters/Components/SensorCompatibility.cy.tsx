import { mount } from '@cypress/react18';

import SensorCompatibility from './SensorCompatibility';
import type { SensorVersionCompatibility } from 'types/cluster.proto';

describe('SensorCompatibility', () => {
    const compatibleVersions = ['4.10.0', '4.11.0', '4.12.0'];

    it('should render "Not applicable" when compatible versions list is empty', () => {
        mount(<SensorCompatibility compatibility="MATCHED" compatibleVersions={[]} />);

        cy.findByTestId('sensorCompatibility').should('contain', 'Not applicable');
    });

    it('should render "Not applicable" when compatible versions is undefined', () => {
        mount(<SensorCompatibility compatibility="MATCHED" />);

        cy.findByTestId('sensorCompatibility').should('contain', 'Not applicable');
    });

    it('should render "Matched" status', () => {
        const compatibility: SensorVersionCompatibility = 'MATCHED';
        mount(
            <SensorCompatibility
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
            />
        );

        cy.findByTestId('sensorCompatibility').should('contain', 'Matched');
    });

    it('should render "Compatible (Behind)" status', () => {
        const compatibility: SensorVersionCompatibility = 'COMPATIBLE_BEHIND';
        mount(
            <SensorCompatibility
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
            />
        );

        cy.findByTestId('sensorCompatibility').should('contain', 'Compatible (Behind)');
    });

    it('should render "Compatible (Ahead)" status', () => {
        const compatibility: SensorVersionCompatibility = 'COMPATIBLE_AHEAD';
        mount(
            <SensorCompatibility
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
            />
        );

        cy.findByTestId('sensorCompatibility').should('contain', 'Compatible (Ahead)');
    });

    it('should render "Incompatible (Behind)" status', () => {
        const compatibility: SensorVersionCompatibility = 'INCOMPATIBLE_BEHIND';
        mount(
            <SensorCompatibility
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
            />
        );

        cy.findByTestId('sensorCompatibility').should('contain', 'Incompatible (Behind)');
    });

    it('should render "Incompatible (Ahead)" status', () => {
        const compatibility: SensorVersionCompatibility = 'INCOMPATIBLE_AHEAD';
        mount(
            <SensorCompatibility
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
            />
        );

        cy.findByTestId('sensorCompatibility').should('contain', 'Incompatible (Ahead)');
    });

    it('should render "Unknown" status', () => {
        const compatibility: SensorVersionCompatibility = 'UNKNOWN';
        mount(
            <SensorCompatibility
                compatibility={compatibility}
                compatibleVersions={compatibleVersions}
            />
        );

        cy.findByTestId('sensorCompatibility').should('contain', 'Unknown');
    });
});
