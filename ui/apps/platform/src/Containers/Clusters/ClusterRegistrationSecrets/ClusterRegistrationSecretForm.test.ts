import { buildRequestData, initialValues } from './ClusterRegistrationSecretForm';
import type { ClusterRegistrationSecretFormValues } from './ClusterRegistrationSecretForm';

describe('buildRequestData', () => {
    const baseFormValues: ClusterRegistrationSecretFormValues = {
        ...initialValues,
        name: 'my-secret',
    };

    it('should return only name when validity is none and max is undefined', () => {
        expect(buildRequestData(baseFormValues)).toEqual({ name: 'my-secret' });
    });

    it('should include validUntil when mode is date', () => {
        const iso = '2050-06-15T23:59:00.000Z';
        expect(
            buildRequestData({ ...baseFormValues, validityMode: 'date', validUntil: iso })
        ).toEqual({
            name: 'my-secret',
            validUntil: iso,
        });
    });

    it('should throw when date mode but value is undefined', () => {
        expect(() =>
            buildRequestData({ ...baseFormValues, validityMode: 'date', validUntil: undefined })
        ).toThrow('A date is required');
    });

    it('should convert hours to seconds duration string', () => {
        expect(
            buildRequestData({ ...baseFormValues, validityMode: 'hours', validFor: '24' })
        ).toEqual({
            name: 'my-secret',
            validFor: '86400s',
        });
    });

    it('should throw when hours mode but value is undefined', () => {
        expect(() =>
            buildRequestData({ ...baseFormValues, validityMode: 'hours', validFor: undefined })
        ).toThrow('An hours value is required');
    });

    it('should include maxRegistrations when a positive number', () => {
        expect(buildRequestData({ ...baseFormValues, maxRegistrations: '5' })).toEqual({
            name: 'my-secret',
            maxRegistrations: '5',
        });
    });

    it('should not include maxRegistrations when undefined', () => {
        expect(buildRequestData({ ...baseFormValues, maxRegistrations: undefined })).toEqual({
            name: 'my-secret',
        });
    });

    it('should combine all fields', () => {
        expect(
            buildRequestData({
                ...baseFormValues,
                validityMode: 'hours',
                validFor: '48',
                maxRegistrations: '10',
            })
        ).toEqual({
            name: 'my-secret',
            validFor: '172800s',
            maxRegistrations: '10',
        });
    });
});
