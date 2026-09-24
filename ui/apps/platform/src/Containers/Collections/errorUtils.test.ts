import { parseConfigError } from './errorUtils';

describe('parseConfigError', () => {
    // Backend messages lead with a lower-case word. getAxiosErrorMessage capitalizes
    // the first character for display, so parseConfigError must classify against the
    // raw message; otherwise these case-sensitive patterns stop matching.
    it('classifies a collection loop and extracts the loopId from a lower-case-leading message', () => {
        const err = new Error(
            "edge between 'aaaaaaaa-1111' and 'bbbbbbbb-2222' would create a loop"
        );

        const result = parseConfigError(err);

        expect(result.type).toBe('CollectionLoop');
        expect(result).toHaveProperty('loopId', 'bbbbbbbb-2222');
    });

    it('classifies a duplicate name on save', () => {
        const err = new Error('collections must have non-empty, unique `name` values');

        expect(parseConfigError(err).type).toBe('DuplicateName');
    });

    it('classifies a duplicate name on update', () => {
        const err = new Error('name already in use');

        expect(parseConfigError(err).type).toBe('DuplicateName');
    });

    it('classifies an empty name', () => {
        const err = new Error('name should not be empty');

        expect(parseConfigError(err).type).toBe('EmptyName');
    });

    it('classifies an invalid rule and shows the capitalized message in details', () => {
        const err = new Error('failed to compile regex "["');

        const result = parseConfigError(err);

        expect(result.type).toBe('InvalidRule');
        expect(result.details).toBe('Failed to compile regex "["');
    });

    it('falls back to UnknownError for an unrecognized message', () => {
        const err = new Error('something entirely unexpected happened');

        const result = parseConfigError(err);

        expect(result.type).toBe('UnknownError');
        expect(result.details).toBe('Something entirely unexpected happened');
    });
});
