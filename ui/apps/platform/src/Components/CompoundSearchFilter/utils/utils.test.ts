import { convertFromInternalToExternalDatePicker } from './utils';

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
    });
});
