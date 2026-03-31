import { isDeploymentStatus } from './deploymentStatus';

describe('isDeploymentStatus', () => {
    it('returns true for valid values', () => {
        expect(isDeploymentStatus('DEPLOYED')).toBe(true);
        expect(isDeploymentStatus('DELETED')).toBe(true);
    });

    it('returns false for invalid values', () => {
        expect(isDeploymentStatus('UNSPECIFIED')).toBe(false);
        expect(isDeploymentStatus('')).toBe(false);
        expect(isDeploymentStatus(null)).toBe(false);
        expect(isDeploymentStatus(undefined)).toBe(false);
        expect(isDeploymentStatus(42)).toBe(false);
    });
});
