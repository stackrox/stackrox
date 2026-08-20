import {
    convertFromInternalToExternalDatePicker,
    serializeAbsoluteDateRange,
    serializeRelativeDateRange,
    serializeRelativeOlderThan,
} from './utils';

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

        it('returns null when either input is not a valid date', () => {
            const validMs = new Date(2025, 0, 1).getTime();
            expect(serializeAbsoluteDateRange(NaN, validMs)).toBeNull();
            expect(serializeAbsoluteDateRange(validMs, NaN)).toBeNull();
        });

        it('returns null when start date is after end date', () => {
            const startMs = new Date(2025, 2, 31).getTime();
            const endMs = new Date(2025, 0, 1).getTime();
            expect(serializeAbsoluteDateRange(startMs, endMs)).toBeNull();
        });

        it('round-trips through convertFromInternalToExternalDatePicker', () => {
            const startMs = new Date(2025, 0, 1).getTime();
            const endMs = new Date(2025, 2, 31).getTime();
            const serialized = serializeAbsoluteDateRange(startMs, endMs);
            expect(serialized).not.toBeNull();
            expect(convertFromInternalToExternalDatePicker(serialized as string)).toEqual(
                'Between Jan 01, 2025 and Mar 31, 2025'
            );
        });
    });

    describe('serializeRelativeOlderThan', () => {
        it('serializes a positive integer', () => {
            expect(serializeRelativeOlderThan(365)).toEqual('>365d');
        });

        it('serializes zero', () => {
            expect(serializeRelativeOlderThan(0)).toEqual('>0d');
        });

        it('returns null for a negative number', () => {
            expect(serializeRelativeOlderThan(-1)).toBeNull();
        });

        it('returns null for a non-integer', () => {
            expect(serializeRelativeOlderThan(1.5)).toBeNull();
        });
    });

    describe('serializeRelativeDateRange', () => {
        it('serializes a valid range', () => {
            expect(serializeRelativeDateRange(30, 90)).toEqual('30d-90d');
        });

        it('serializes equal min and max', () => {
            expect(serializeRelativeDateRange(7, 7)).toEqual('7d-7d');
        });

        it('serializes zero values', () => {
            expect(serializeRelativeDateRange(0, 30)).toEqual('0d-30d');
        });

        it('returns null when min exceeds max', () => {
            expect(serializeRelativeDateRange(90, 30)).toBeNull();
        });

        it('returns null for a negative number', () => {
            expect(serializeRelativeDateRange(-1, 30)).toBeNull();
            expect(serializeRelativeDateRange(0, -1)).toBeNull();
        });

        it('returns null for a non-integer', () => {
            expect(serializeRelativeDateRange(1.5, 30)).toBeNull();
            expect(serializeRelativeDateRange(0, 2.5)).toBeNull();
        });
    });
});
