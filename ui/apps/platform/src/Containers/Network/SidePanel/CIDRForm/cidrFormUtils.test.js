import { getHasDuplicateCIDRNames, getHasDuplicateCIDRAddresses } from './cidrFormUtils';

describe('cidrFormUtils', () => {
    describe('getHasDuplicateCIDRNames', () => {
        it('should have duplicate CIDR names', () => {
            const values = {
                entities: [
                    { entity: { name: 'Name 1' } },
                    { entity: { name: 'Name 2' } },
                    { entity: { name: 'Name 1' } },
                ],
            };
            const hasDuplicateCIDRNames = getHasDuplicateCIDRNames(values);
            expect(hasDuplicateCIDRNames).toEqual(true);
        });

        it('should not have duplicate CIDR names', () => {
            const values = {
                entities: [{ entity: { name: 'Name 1' } }, { entity: { name: 'Name 2' } }],
            };
            const hasDuplicateCIDRNames = getHasDuplicateCIDRNames(values);
            expect(hasDuplicateCIDRNames).toEqual(false);
        });
    });

    describe('getHasDuplicateCIDRAddresses', () => {
        it('should have duplicate CIDR addresses', () => {
            const values = {
                entities: [
                    { entity: { cidr: '104.0.0.0/8' } },
                    { entity: { cidr: '192.2.2.2/24' } },
                    { entity: { cidr: '104.0.0.0/8' } },
                ],
            };
            const hasDuplicateCIDRAddresses = getHasDuplicateCIDRAddresses(values);
            expect(hasDuplicateCIDRAddresses).toEqual(true);
        });

        it('should not have duplicate CIDR addresses', () => {
            const values = {
                entities: [
                    { entity: { cidr: '104.0.0.0/8' } },
                    { entity: { cidr: '192.2.2.2/24' } },
                    { entity: { cidr: '114.2.0.0/8' } },
                ],
            };
            const hasDuplicateCIDRAddresses = getHasDuplicateCIDRAddresses(values);
            expect(hasDuplicateCIDRAddresses).toEqual(false);
        });
    });
});
