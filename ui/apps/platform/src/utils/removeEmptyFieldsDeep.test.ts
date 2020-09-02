import removeEmptyFieldsDeep from './removeEmptyFieldsDeep';

describe('removeEmptyFieldsDeep', () => {
    it('removes first-level empty fields', () => {
        const result = removeEmptyFieldsDeep({
            key1: null,
            key2: [],
            key3: undefined,
            key4: '',
            key5: {},
            key6: new Set(),
            key7: 'value',
            key8: 0,
        });
        expect(result).toStrictEqual({
            key7: 'value',
            key8: 0,
        });
    });

    it('removes empty fields deep', () => {
        const result = removeEmptyFieldsDeep({
            objKey: {
                objKey: {
                    key1: null,
                    key2: [],
                    key3: undefined,
                    key4: '',
                    key5: {},
                    key6: new Set(),
                    key7: 'value',
                    key8: 0,
                },
            },
        });
        expect(result).toStrictEqual({
            objKey: {
                objKey: {
                    key7: 'value',
                    key8: 0,
                },
            },
        });
    });

    it('removes empty objects with only empty fields', () => {
        const result = removeEmptyFieldsDeep({
            objKey: {
                key1: null,
            },
            key: 'value',
        });
        expect(result).toStrictEqual({
            key: 'value',
        });
    });

    it('handles an empty object', () => {
        const result = removeEmptyFieldsDeep({});
        expect(result).toStrictEqual({});
    });

    it('does not mutate the original object', () => {
        const obj = {
            key1: null,
            key2: 'value',
        };
        const result = removeEmptyFieldsDeep(obj);
        expect(result).toStrictEqual({ key2: 'value' });
        expect(obj.key1).toBeNull();
    });

    it('deeply clones the original object', () => {
        const obj = {
            key1: {
                key2: 'value',
            },
        };
        const result = removeEmptyFieldsDeep(obj);
        expect(result).toStrictEqual(obj);

        expect(result).not.toBe(obj);
        expect(obj.key1).not.toBe((result as typeof obj).key1);
    });
});
