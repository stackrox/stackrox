import { getQueryObject } from './queryStringUtils';

describe('getQueryObject', () => {
    /**
     * This test ensures `getQueryObject` correctly parses url filters up to the `arrayLimit` of 200 elements.
     * Beyond this limit, the result becomes an object, which `parseFilter` inside `useURLSearch` does not handle properly.
     */
    it('should parse url array filters up to the arrayLimit of 200 elements', () => {
        const numElements = 200;
        let largeArrayQuery = '?';

        for (let i = 0; i < numElements; i += 1) {
            largeArrayQuery += `Namespace%20ID[${i}]=${i}&`;
        }

        const result = getQueryObject(largeArrayQuery);

        expect(Array.isArray(result['Namespace ID'])).toBe(true);

        expect(result['Namespace ID']?.[0] ?? 'undefined').toBe('0');
        expect(result['Namespace ID']?.[numElements - 1] ?? 'undefined').toBe(`${numElements - 1}`);
    });
});
