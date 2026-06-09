import { convertFromInternalToExternalDatePicker, serializeAbsoluteDateRange } from './utils';

describe('utils', () => {
    describe('convertFromInternalToExternalDatePicker', () => {
        it('formats "after" condition correctly', () => {
            expect(convertFromInternalToExternalDatePicker('>2024-01-01')).toEqual(
                'After Jan 01, 2024'
            );
        });

        it('formats "before" condition correctly', () => {
            expect(convertFromInternalToExternalDatePicker('<2024-12-31')).toEqual(
                'Before Dec 31, 2024'
            );
        });

        it('formats date without condition as "on"', () => {
            expect(convertFromInternalToExternalDatePicker('2024-06-15')).toEqual(
                'On Jun 15, 2024'
            );
        });

        it('returns original value for invalid date', () => {
            expect(convertFromInternalToExternalDatePicker('>invalid-date')).toEqual(
                '>invalid-date'
            );
        });

        it('returns original value for empty string', () => {
            expect(convertFromInternalToExternalDatePicker('')).toEqual('');
        });

        it('handles multiple condition characters', () => {
            expect(convertFromInternalToExternalDatePicker('>>2024-01-01')).toEqual(
                'After Jan 01, 2024'
            );
        });

        it('does not throw errors and returns original value for malformed input', () => {
            expect(() => convertFromInternalToExternalDatePicker('>2024-13-50')).not.toThrow();
            expect(convertFromInternalToExternalDatePicker('>2024-13-50')).toEqual('>2024-13-50');
        });

        it('formats an absolute date range correctly', () => {
            const startMs = new Date(2025, 0, 1).getTime();
            const endMs = new Date(2025, 2, 31).getTime();
            expect(convertFromInternalToExternalDatePicker(`tr/${startMs}-${endMs}`)).toEqual(
                'Between Jan 01, 2025 and Mar 31, 2025'
            );
        });

        it('formats a relative date range correctly', () => {
            expect(convertFromInternalToExternalDatePicker('30d-90d')).toEqual(
                'Between 30 and 90 days ago'
            );
        });

        it('formats an open-ended relative range correctly', () => {
            expect(convertFromInternalToExternalDatePicker('>365d')).toEqual(
                'More than 365 days ago'
            );
        });

        it('returns original value for malformed absolute range', () => {
            expect(convertFromInternalToExternalDatePicker('tr/abc-def')).toEqual('tr/abc-def');
            expect(convertFromInternalToExternalDatePicker('tr/123')).toEqual('tr/123');
            expect(convertFromInternalToExternalDatePicker('tr/')).toEqual('tr/');
        });
    });

    describe('serializeAbsoluteDateRange', () => {
        it('serializes to tr/<startMs>-<endMs> with start-of-day and end-of-day boundaries', () => {
            const startMs = new Date(2025, 0, 1, 10, 30).getTime();
            const endMs = new Date(2025, 2, 31, 8, 0).getTime();
            const expectedStartMs = new Date(2025, 0, 1, 0, 0, 0, 0).getTime();
            const expectedEndMs = new Date(2025, 2, 31, 23, 59, 59, 999).getTime();
            expect(serializeAbsoluteDateRange(startMs, endMs)).toEqual(
                `tr/${expectedStartMs}-${expectedEndMs}`
            );
        });

        it('serializes a same-day range spanning the whole day', () => {
            const dayMs = new Date(2025, 5, 15, 12, 0).getTime();
            const expectedStartMs = new Date(2025, 5, 15, 0, 0, 0, 0).getTime();
            const expectedEndMs = new Date(2025, 5, 15, 23, 59, 59, 999).getTime();
            expect(serializeAbsoluteDateRange(dayMs, dayMs)).toEqual(
                `tr/${expectedStartMs}-${expectedEndMs}`
            );
        });

        it('throws when either input is not a valid date', () => {
            const validMs = new Date(2025, 0, 1).getTime();
            expect(() => serializeAbsoluteDateRange(NaN, validMs)).toThrow();
            expect(() => serializeAbsoluteDateRange(validMs, NaN)).toThrow();
        });

        it('throws when start date is after end date', () => {
            const startMs = new Date(2025, 2, 31).getTime();
            const endMs = new Date(2025, 0, 1).getTime();
            expect(() => serializeAbsoluteDateRange(startMs, endMs)).toThrow();
        });

        it('round-trips through convertFromInternalToExternalDatePicker', () => {
            const startMs = new Date(2025, 0, 1).getTime();
            const endMs = new Date(2025, 2, 31).getTime();
            expect(
                convertFromInternalToExternalDatePicker(serializeAbsoluteDateRange(startMs, endMs))
            ).toEqual('Between Jan 01, 2025 and Mar 31, 2025');
        });
    });
});
