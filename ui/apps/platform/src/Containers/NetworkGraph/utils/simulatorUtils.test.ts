import { getSimulationPanelHeaderText } from './simulatorUtils';

describe('getSimulationPanelHeaderText', () => {
    it('should return the text that describes the scope of the generated network policies', () => {
        const cluster = { id: '1', name: 'production' };

        expect(getSimulationPanelHeaderText({ cluster, namespaces: [], deployments: [] })).toEqual(
            'Simulate network policies for all deployments in cluster "production"'
        );
        expect(
            getSimulationPanelHeaderText({ cluster, namespaces: ['payments'], deployments: [] })
        ).toEqual(
            'Simulate network policies for all deployments in namespace "payments" in cluster "production"'
        );

        expect(
            getSimulationPanelHeaderText({
                cluster,
                namespaces: ['payments', 'backend', 'frontend'],
                deployments: [],
            })
        ).toEqual(
            'Simulate network policies for all deployments in namespaces "payments", "backend", "frontend" in cluster "production"'
        );

        expect(
            getSimulationPanelHeaderText({
                cluster,
                namespaces: ['payments', 'backend', 'frontend', 'too-many'],
                deployments: [],
            })
        ).toEqual(
            'Simulate network policies for all deployments in 4 namespaces in cluster "production"'
        );

        expect(
            getSimulationPanelHeaderText({
                cluster,
                namespaces: ['payments'],
                deployments: ['visa-processor'],
            })
        ).toEqual(
            'Simulate network policies for deployment "payments/visa-processor" in cluster "production"'
        );

        expect(
            getSimulationPanelHeaderText({
                cluster,
                namespaces: ['payments'],
                deployments: ['visa-processor', 'gateway'],
            })
        ).toEqual(
            'Simulate network policies for deployments "payments/visa-processor", "payments/gateway" in cluster "production"'
        );

        // Note we don't scope deployment names to a single namespace, so the query result is
        // the combination of "namespaces x deployments"
        expect(
            getSimulationPanelHeaderText({
                cluster,
                namespaces: ['payments', 'backend'],
                deployments: ['visa-processor', 'gateway'],
            })
        ).toEqual('Simulate network policies for 4 deployments in cluster "production"');
    });
});
