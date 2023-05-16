import { checkArrayContainsArray } from './arrayUtils';

describe('arrayUtils', () => {
    describe('checkArrayContainsArray', () => {
        it('should return false when no overlap in arrays', () => {
            const allowed = ['BUILD', 'DEPLOY'];
            const candidate = ['RUNTIME'];

            const doesContain = checkArrayContainsArray(allowed, candidate);

            expect(doesContain).toBe(false);
        });

        it('should return false when only partial overlap in arrays', () => {
            const allowed = ['BUILD', 'DEPLOY'];
            const candidate = ['BUILD', 'RUNTIME'];

            const doesContain = checkArrayContainsArray(allowed, candidate);

            expect(doesContain).toBe(false);
        });

        it('should return true when when single-element arrays for both args overlap', () => {
            const allowed = ['RUNTIME'];
            const candidate = ['RUNTIME'];

            const doesContain = checkArrayContainsArray(allowed, candidate);

            expect(doesContain).toBe(true);
        });

        it('should return true when when multi-element array is contained allowed array', () => {
            const allowed = ['BUILD', 'DEPLOY', 'RUNTIME'];
            const candidate = ['DEPLOY', 'RUNTIME'];

            const doesContain = checkArrayContainsArray(allowed, candidate);

            expect(doesContain).toBe(true);
        });
    });
});
