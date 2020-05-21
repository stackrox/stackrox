import { parseValueStr, formatValueStr } from './utils';

describe('policyFormUtils', () => {
    describe('parseValueStr', () => {
        it('should parse Environment Variable values', () => {
            const source = 'source1';
            const key = 'key1';
            const value = 'value1';
            const valueObj = parseValueStr(`${source}=${key}=${value}`);
            expect(valueObj.source).toEqual(source);
            expect(valueObj.key).toEqual(key);
            expect(valueObj.value).toEqual(value);
        });

        it('should parse nested policy criteria values', () => {
            const key = 'key1';
            const value = 'value1';
            const valueObj = parseValueStr(`${key}=${value}`);
            expect(valueObj.key).toEqual(key);
            expect(valueObj.value).toEqual(value);
            expect(valueObj.source).toBeUndefined();
        });

        it('should parse flat policy criteria values', () => {
            const value = 'value1';
            const valueObj = parseValueStr(value);
            expect(valueObj.key).toBeUndefined();
            expect(valueObj.value).toEqual(value);
            expect(valueObj.source).toBeUndefined();
        });

        it('should not parse empty policy criteria value', () => {
            const value = '';
            const valueObj = parseValueStr(value);
            expect(valueObj.key).toBeUndefined();
            expect(valueObj.value).toEqual(value);
            expect(valueObj.source).toBeUndefined();
        });

        it('should not parse non-valid policy criteria values', () => {
            const value = `hello=thisis=aninvalid=value`;
            const valueObj = parseValueStr(value);
            expect(valueObj.key).toBeUndefined();
            expect(valueObj.value).toEqual(value);
            expect(valueObj.source).toBeUndefined();
        });
    });

    describe('formatValueStr', () => {
        it('should format Environment Variable value to value obj with value string', () => {
            const source = 'source1';
            const key = 'key1';
            const value = 'value1';
            const valueObj = {
                source,
                key,
                value,
            };
            const valueStr = formatValueStr(valueObj);
            expect(valueStr).toBe(`${source}=${key}=${value}`);
        });

        it('should format nested policy criteria values to value obj with value string', () => {
            const key = 'key1';
            const value = 'value1';
            const valueObj = {
                key,
                value,
            };
            const valueStr = formatValueStr(valueObj);
            expect(valueStr).toBe(`${key}=${value}`);
        });

        it('should format flat policy criteria values to value obj with value string', () => {
            const value = 'value1';
            const valueObj = {
                value,
            };
            const valueStr = formatValueStr(valueObj);
            expect(valueStr).toBe(value);
        });
    });
});
