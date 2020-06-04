import { parseValueStr, formatValueStr, parseNumericComparisons } from './utils';

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

        describe('parsing numeric comparison fields', () => {
            it('should parse a numeric field with no space between operator and number', () => {
                const value = `>=8`;
                const field = 'CVSS';

                const valueObj = parseValueStr(value, field);

                expect(valueObj.key).toEqual('>=');
                expect(valueObj.value).toEqual('8');
                expect(valueObj.source).toBeUndefined();
            });

            it('should parse a numeric field with a space between operator and number', () => {
                const value = `>= 8`;
                const field = 'CVSS';

                const valueObj = parseValueStr(value, field);

                expect(valueObj.key).toEqual('>=');
                expect(valueObj.value).toEqual('8');
                expect(valueObj.source).toBeUndefined();
            });

            it('should parse a numeric field with no space or operator, and number, as equals', () => {
                const value = `7.5`;
                const field = 'CVSS';

                const valueObj = parseValueStr(value, field);

                expect(valueObj.key).toEqual('=');
                expect(valueObj.value).toEqual('7.5');
                expect(valueObj.source).toBeUndefined();
            });

            it('should parse a numeric field with a space but no operator, and number, as equals', () => {
                const value = ` 7.5`;
                const field = 'CVSS';

                const valueObj = parseValueStr(value, field);

                expect(valueObj.key).toEqual('=');
                expect(valueObj.value).toEqual('7.5');
                expect(valueObj.source).toBeUndefined();
            });
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

        it('should format Environment Variable empty value to value obj with value string', () => {
            const source = 'source1';
            const key = 'key1';
            const value = '';
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

    /**
     * Handling these cases:
     *  ">7"  ==> {  key: '>', value: '7.5' }
     *  "> 7"  ==> {  key: '>', value: '7.5' }
     *  ">=7"  ==> {  key: '>=', value: '7.5' }
     *  ">= 7"  ==> {  key: '>=', value: '7.5' }
     *  "7"  ==> {  key: '=', value: '7.5' }
     *  " 7"  ==> {  key: '=', value: '7.5' }
     *  "<7"  ==> {  key: '<', value: '7.5' }
     *  "< 7"  ==> {  key: '<', value: '7.5' }
     *  "<=7"  ==> {  key: '<=', value: '.5' }
     *  "<= 7"  ==> {  key: '<=', value: '.5' }
     */
    describe('parseNumericComparisons', () => {
        it('should parse an Integer greater-than comparison (no space)', () => {
            const value = '>7';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('>');
            expect(num).toEqual('7');
        });

        it('should parse an Integer greater-than comparison (with space)', () => {
            const value = '> 7';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('>');
            expect(num).toEqual('7');
        });

        it('should parse a Float greater-than comparison (no space)', () => {
            const value = '>8.5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('>');
            expect(num).toEqual('8.5');
        });

        it('should parse a Float greater-than comparison (with space)', () => {
            const value = '> 8.25';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('>');
            expect(num).toEqual('8.25');
        });

        it('should parse an Integer greater-than-or-equal comparison (no space)', () => {
            const value = '>=5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('>=');
            expect(num).toEqual('5');
        });

        it('should parse an Integer greater-than-or-equal comparison (with space)', () => {
            const value = '>= 5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('>=');
            expect(num).toEqual('5');
        });

        it('should parse a Float greater-than-or-equal comparison (no space)', () => {
            const value = '>=1.5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('>=');
            expect(num).toEqual('1.5');
        });

        it('should parse a Float greater-than-or-equal comparison (with space)', () => {
            const value = '>= 1.5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('>=');
            expect(num).toEqual('1.5');
        });

        it('should parse an Integer equal comparison (no space)', () => {
            const value = '5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toBeFalsy();
            expect(num).toEqual('5');
        });

        it('should parse an Integer equal comparison (with space)', () => {
            const value = ' 5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toBeFalsy();
            expect(num).toEqual('5');
        });

        it('should parse a Float equal comparison (no space)', () => {
            const value = '1.5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toBeFalsy();
            expect(num).toEqual('1.5');
        });

        it('should parse a Float equal comparison (with space)', () => {
            const value = ' 1.5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toBeFalsy();
            expect(num).toEqual('1.5');
        });

        it('should parse an Integer less-than comparison (no space)', () => {
            const value = '<7';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('<');
            expect(num).toEqual('7');
        });

        it('should parse an Integer less-than comparison (with space)', () => {
            const value = '< 7';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('<');
            expect(num).toEqual('7');
        });

        it('should parse a Float less-than comparison (no space)', () => {
            const value = '<8.5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('<');
            expect(num).toEqual('8.5');
        });

        it('should parse a Float less-than comparison (with space)', () => {
            const value = '< 8.25';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('<');
            expect(num).toEqual('8.25');
        });

        it('should parse an Integer less-than-or-equal comparison (no space)', () => {
            const value = '<=5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('<=');
            expect(num).toEqual('5');
        });

        it('should parse an Integer less-than-or-equal comparison (with space)', () => {
            const value = '<= 5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('<=');
            expect(num).toEqual('5');
        });

        it('should parse a Float less-than-or-equal comparison (no space)', () => {
            const value = '<=1.5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('<=');
            expect(num).toEqual('1.5');
        });

        it('should parse a Float less-than-or-equal comparison (with space)', () => {
            const value = '<= 1.5';

            const [comparison, num] = parseNumericComparisons(value);

            expect(comparison).toEqual('<=');
            expect(num).toEqual('1.5');
        });
    });
});
