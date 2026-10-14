import { parseConfigError } from './errorUtils';

describe('parseConfigError', () => {
    it('classifies a collection loop and extracts the loopId from a lower-case-leading message', () => {
        const err = new Error(
            "edge between 'aaaaaaaa-1111' and 'bbbbbbbb-2222' would create a loop"
        );

        const result = parseConfigError(err);

        expect(result.type).toBe('CollectionLoop');
        expect(result).toHaveProperty('loopId', 'bbbbbbbb-2222');
    });

    it('classifies an invalid rule and preserves the backend message in details', () => {
        const err = new Error('failed to compile regex "["');

        const result = parseConfigError(err);

        expect(result.type).toBe('InvalidRule');
        expect(result.details).toBe('failed to compile regex "["');
    });

    it('falls back to UnknownError for an unrecognized message', () => {
        const err = new Error('something entirely unexpected happened');

        const result = parseConfigError(err);

        expect(result.type).toBe('UnknownError');
        expect(result.details).toBe('something entirely unexpected happened');
    });
});
