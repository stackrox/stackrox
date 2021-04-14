import { AddedBaseline, RemovedBaseline, UnmodifiedBaseline } from '../baselineSimulationTypes';
import getPropertiesByStatus from './getPropertiesByStatus';

describe('getPropertiesByStatus', () => {
    it('should return the properties of an added baseline', () => {
        const datum = {
            peer: {
                entity: {
                    id: '12345',
                    name: 'deployment',
                    namespace: 'stackrox',
                    type: 'DEPLOYMENT',
                },
                added: {
                    port: '3000',
                    protocol: 'L4_PROTOCOL_TCP',
                    ingress: true,
                },
                state: 'active',
            },
            simulatedStatus: 'ADDED',
        } as AddedBaseline;
        expect(getPropertiesByStatus(datum)).toEqual({
            port: '3000',
            protocol: 'L4_PROTOCOL_TCP',
            ingress: true,
        });
    });

    it('should return the properties of a removed baseline', () => {
        const datum = {
            peer: {
                entity: {
                    id: '12345',
                    name: 'deployment',
                    namespace: 'stackrox',
                    type: 'DEPLOYMENT',
                },
                removed: {
                    port: '4000',
                    protocol: 'L4_PROTOCOL_UDP',
                    ingress: false,
                },
                state: 'active',
            },
            simulatedStatus: 'REMOVED',
        } as RemovedBaseline;
        expect(getPropertiesByStatus(datum)).toEqual({
            port: '4000',
            protocol: 'L4_PROTOCOL_UDP',
            ingress: false,
        });
    });

    it('should return the properties of an unmodified baseline', () => {
        const datum = {
            peer: {
                entity: {
                    id: '12345',
                    name: 'deployment',
                    namespace: 'stackrox',
                    type: 'DEPLOYMENT',
                },
                unmodified: {
                    port: '5000',
                    protocol: 'L4_PROTOCOL_UDP',
                    ingress: true,
                },
                state: 'active',
            },
            simulatedStatus: 'UNMODIFIED',
        } as UnmodifiedBaseline;
        expect(getPropertiesByStatus(datum)).toEqual({
            port: '5000',
            protocol: 'L4_PROTOCOL_UDP',
            ingress: true,
        });
    });
});
