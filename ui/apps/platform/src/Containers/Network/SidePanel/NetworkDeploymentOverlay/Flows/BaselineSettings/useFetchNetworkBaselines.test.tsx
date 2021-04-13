import { getPeerEntityName } from './useFetchNetworkBaselines';

describe('useFetchNetworkBaselines', () => {
    describe('getPeerEntityName', () => {
        it('should get the name from a network baseline peer with an entity type of DEPLOYMENT', () => {
            const peer = {
                entity: {
                    info: {
                        deployment: {
                            name: 'deployment-1',
                        },
                        type: 'DEPLOYMENT',
                    },
                },
            };
            expect(getPeerEntityName(peer)).toEqual('deployment-1');
        });

        it('should get the name from a network baseline peer with an entity type of INTERNET', () => {
            const peer = {
                entity: {
                    info: {
                        type: 'INTERNET',
                    },
                },
            };
            expect(getPeerEntityName(peer)).toEqual('External Entities');
        });

        it('should get the name from a network baseline peer with an entity type of EXTERNAL_SOURCE', () => {
            const peer = {
                entity: {
                    info: {
                        type: 'EXTERNAL_SOURCE',
                        externalSource: {
                            name: 'cloud-1',
                        },
                    },
                },
            };
            expect(getPeerEntityName(peer)).toEqual('cloud-1');
        });
    });
});
