import { truncate } from './textUtils';

describe('truncate pipe', () => {
    it('should return the same string if shorter than length', () => {
        const str = 'The quick brown fox jumps over the lazy dog.';
        const maxLength = 45;

        const truncatedStr = truncate(str, maxLength);

        expect(truncatedStr).toEqual(str);
    });

    it('should reduce the string to length given', () => {
        const str = 'The quick brown fox jumps over the lazy dog.';
        const maxLength = 15;

        const truncatedStr = truncate(str, maxLength);

        expect(truncatedStr).toEqual('The quick brown…');
    });

    it('should reduce the string to the closet word boundary within length given', () => {
        const str = 'The quick brown fox jumps over the lazy dog.';
        const maxLength = 14;

        const truncatedStr = truncate(str, maxLength);

        expect(truncatedStr).toEqual('The quick…');
    });
});
