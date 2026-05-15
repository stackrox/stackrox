import { ruleFieldValueMapper, searchFieldValueMapper } from './entityScopeRules';

describe('entityScopeRules', () => {
    describe('searchFieldValueMapper', () => {
        it('maps a quoted string to EXACT match with unquoted value', () => {
            expect(searchFieldValueMapper('"production"')).toEqual({
                matchType: 'EXACT',
                value: 'production',
            });
        });

        it('maps an unquoted string to REGEX match with raw value', () => {
            expect(searchFieldValueMapper('prod.*')).toEqual({
                matchType: 'REGEX',
                value: 'prod.*',
            });
        });

        it('does not treat a single-quote-delimited string as exact', () => {
            expect(searchFieldValueMapper("'production'")).toEqual({
                matchType: 'REGEX',
                value: "'production'",
            });
        });

        it('handles quoted string with internal quotes', () => {
            expect(searchFieldValueMapper('"my"cluster"')).toEqual({
                matchType: 'EXACT',
                value: 'my"cluster',
            });
        });

        it('handles empty quoted string', () => {
            expect(searchFieldValueMapper('""')).toEqual({
                matchType: 'EXACT',
                value: '',
            });
        });

        it('handles key=value label format as REGEX', () => {
            expect(searchFieldValueMapper('app=frontend')).toEqual({
                matchType: 'REGEX',
                value: 'app=frontend',
            });
        });

        it('handles quoted key=value label format as EXACT', () => {
            expect(searchFieldValueMapper('"app=frontend"')).toEqual({
                matchType: 'EXACT',
                value: 'app=frontend',
            });
        });
    });

    describe('ruleFieldValueMapper', () => {
        it('wraps EXACT match value in quotes for SearchFilter convention', () => {
            expect(ruleFieldValueMapper({ matchType: 'EXACT', value: 'production' })).toBe(
                '"production"'
            );
        });

        it('returns REGEX match value as-is without r/ prefix', () => {
            expect(ruleFieldValueMapper({ matchType: 'REGEX', value: 'prod.*' })).toBe('prod.*');
        });

        it('does not escape internal quotes in EXACT match values', () => {
            expect(ruleFieldValueMapper({ matchType: 'EXACT', value: 'my"cluster' })).toBe(
                '"my"cluster"'
            );
        });

        it('returns empty string for REGEX with empty value', () => {
            expect(ruleFieldValueMapper({ matchType: 'REGEX', value: '' })).toBe('');
        });

        it('handles EXACT with key=value label format', () => {
            expect(ruleFieldValueMapper({ matchType: 'EXACT', value: 'app=frontend' })).toBe(
                '"app=frontend"'
            );
        });

        it('handles REGEX with key=value label format', () => {
            expect(ruleFieldValueMapper({ matchType: 'REGEX', value: 'app=front.*' })).toBe(
                'app=front.*'
            );
        });
    });

    describe('searchFieldValueMapper and ruleFieldValueMapper round-trip', () => {
        it('round-trips an EXACT value', () => {
            const original = '"production"';
            const ruleValue = searchFieldValueMapper(original);
            const result = ruleFieldValueMapper(ruleValue);
            expect(result).toBe(original);
        });

        it('round-trips a REGEX value', () => {
            const original = 'prod.*';
            const ruleValue = searchFieldValueMapper(original);
            const result = ruleFieldValueMapper(ruleValue);
            expect(result).toBe(original);
        });

        it('round-trips a key=value label', () => {
            const original = 'app=frontend';
            const ruleValue = searchFieldValueMapper(original);
            const result = ruleFieldValueMapper(ruleValue);
            expect(result).toBe(original);
        });

        it('round-trips a quoted key=value label', () => {
            const original = '"app=frontend"';
            const ruleValue = searchFieldValueMapper(original);
            const result = ruleFieldValueMapper(ruleValue);
            expect(result).toBe(original);
        });
    });
});
