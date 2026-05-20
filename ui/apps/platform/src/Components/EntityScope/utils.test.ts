import type { EntityScopeRule } from 'services/ReportsService.types';
import {
    getEntityScopeRulesFromSearchFilterForClusterNamespaceDeployment,
    getSearchFilterFromEntityScopeRules,
    getSearchFilterWithoutEntityScope,
    ruleValueToSearchValue,
    searchFieldLabelMapForClusterNamespaceDeployment,
    searchValueToRuleValue,
} from './utils';

describe('EntityScope utils', () => {
    describe('searchValueToRuleValue', () => {
        it('maps a quoted string to EXACT match with unquoted value', () => {
            expect(searchValueToRuleValue('"production"')).toEqual({
                matchType: 'EXACT',
                value: 'production',
            });
        });

        it('maps an unquoted string to REGEX match with raw value', () => {
            expect(searchValueToRuleValue('prod.*')).toEqual({
                matchType: 'REGEX',
                value: 'prod.*',
            });
        });

        it('does not treat a single-quote-delimited string as exact', () => {
            expect(searchValueToRuleValue("'production'")).toEqual({
                matchType: 'REGEX',
                value: "'production'",
            });
        });

        it('handles quoted string with internal quotes', () => {
            expect(searchValueToRuleValue('"my"cluster"')).toEqual({
                matchType: 'EXACT',
                value: 'my"cluster',
            });
        });

        it('handles empty quoted string', () => {
            expect(searchValueToRuleValue('""')).toEqual({
                matchType: 'EXACT',
                value: '',
            });
        });

        it('handles key=value label format as REGEX', () => {
            expect(searchValueToRuleValue('app=frontend')).toEqual({
                matchType: 'REGEX',
                value: 'app=frontend',
            });
        });

        it('handles quoted key=value label format as EXACT', () => {
            expect(searchValueToRuleValue('"app=frontend"')).toEqual({
                matchType: 'EXACT',
                value: 'app=frontend',
            });
        });
    });

    describe('ruleValueToSearchValue', () => {
        it('wraps EXACT match value in quotes for SearchFilter convention', () => {
            expect(ruleValueToSearchValue({ matchType: 'EXACT', value: 'production' })).toBe(
                '"production"'
            );
        });

        it('returns REGEX match value as-is without r/ prefix', () => {
            expect(ruleValueToSearchValue({ matchType: 'REGEX', value: 'prod.*' })).toBe('prod.*');
        });

        it('does not escape internal quotes in EXACT match values', () => {
            expect(ruleValueToSearchValue({ matchType: 'EXACT', value: 'my"cluster' })).toBe(
                '"my"cluster"'
            );
        });

        it('returns empty string for REGEX with empty value', () => {
            expect(ruleValueToSearchValue({ matchType: 'REGEX', value: '' })).toBe('');
        });

        it('handles EXACT with key=value label format', () => {
            expect(ruleValueToSearchValue({ matchType: 'EXACT', value: 'app=frontend' })).toBe(
                '"app=frontend"'
            );
        });

        it('handles REGEX with key=value label format', () => {
            expect(ruleValueToSearchValue({ matchType: 'REGEX', value: 'app=front.*' })).toBe(
                'app=front.*'
            );
        });
    });

    describe('getSearchFilterFromEntityScopeRules', () => {
        const map = searchFieldLabelMapForClusterNamespaceDeployment;

        it('converts EXACT rule to quoted SearchFilter value', () => {
            const result = getSearchFilterFromEntityScopeRules(
                [
                    {
                        entity: 'SCOPE_ENTITY_CLUSTER',
                        field: 'FIELD_NAME',
                        values: [{ matchType: 'EXACT', value: 'production' }],
                    },
                ],
                map
            );
            expect(result).toEqual({ Cluster: ['"production"'] });
        });

        it('converts REGEX rule to unquoted SearchFilter value', () => {
            const result = getSearchFilterFromEntityScopeRules(
                [
                    {
                        entity: 'SCOPE_ENTITY_CLUSTER',
                        field: 'FIELD_NAME',
                        values: [{ matchType: 'REGEX', value: 'prod.*' }],
                    },
                ],
                map
            );
            expect(result).toEqual({ Cluster: ['prod.*'] });
        });

        it('merges multiple values into a single SearchFilter key', () => {
            const result = getSearchFilterFromEntityScopeRules(
                [
                    {
                        entity: 'SCOPE_ENTITY_CLUSTER',
                        field: 'FIELD_NAME',
                        values: [
                            { matchType: 'EXACT', value: 'production' },
                            { matchType: 'REGEX', value: 'staging.*' },
                        ],
                    },
                ],
                map
            );
            expect(result).toEqual({ Cluster: ['"production"', 'staging.*'] });
        });

        it('converts rules across all entity types', () => {
            const result = getSearchFilterFromEntityScopeRules(
                [
                    {
                        entity: 'SCOPE_ENTITY_CLUSTER',
                        field: 'FIELD_NAME',
                        values: [{ matchType: 'EXACT', value: 'prod' }],
                    },
                    {
                        entity: 'SCOPE_ENTITY_NAMESPACE',
                        field: 'FIELD_NAME',
                        values: [{ matchType: 'REGEX', value: 'default.*' }],
                    },
                    {
                        entity: 'SCOPE_ENTITY_DEPLOYMENT',
                        field: 'FIELD_LABEL',
                        values: [{ matchType: 'EXACT', value: 'app=web' }],
                    },
                ],
                map
            );
            expect(result).toEqual({
                Cluster: ['"prod"'],
                Namespace: ['default.*'],
                'Deployment Label': ['"app=web"'],
            });
        });

        it('accumulates values from multiple rules for the same field', () => {
            const result = getSearchFilterFromEntityScopeRules(
                [
                    {
                        entity: 'SCOPE_ENTITY_CLUSTER',
                        field: 'FIELD_NAME',
                        values: [{ matchType: 'EXACT', value: 'prod' }],
                    },
                    {
                        entity: 'SCOPE_ENTITY_CLUSTER',
                        field: 'FIELD_NAME',
                        values: [{ matchType: 'REGEX', value: 'staging.*' }],
                    },
                ],
                map
            );
            expect(result).toEqual({ Cluster: ['"prod"', 'staging.*'] });
        });

        it('ignores rules for entities not in the map', () => {
            const result = getSearchFilterFromEntityScopeRules(
                [
                    {
                        entity: 'SCOPE_ENTITY_NAMESPACE',
                        field: 'FIELD_NAME',
                        values: [{ matchType: 'REGEX', value: 'default' }],
                    },
                ],
                { Cluster: { entity: 'SCOPE_ENTITY_CLUSTER', field: 'FIELD_NAME' } }
            );
            expect(result).toEqual({});
        });

        it('returns empty object for empty rules', () => {
            expect(getSearchFilterFromEntityScopeRules([], map)).toEqual({});
        });
    });

    describe('getSearchFilterWithoutEntityScope', () => {
        it('removes all entity scope fields from SearchFilter', () => {
            const result = getSearchFilterWithoutEntityScope({
                Cluster: '"production"',
                Namespace: 'default',
                Deployment: 'nginx',
                Severity: 'Critical',
                CVE: 'CVE-2024-1234',
            });
            expect(result).toEqual({
                Severity: 'Critical',
                CVE: 'CVE-2024-1234',
            });
        });

        it('removes label, annotation, and ID fields', () => {
            const result = getSearchFilterWithoutEntityScope({
                'Cluster Label': 'env=prod',
                'Namespace Annotation': 'owner=sre',
                'Deployment Label': 'app=web',
                'Cluster ID': 'abc',
                'Namespace ID': 'def',
                'Deployment ID': 'ghi',
                Fixable: 'true',
            });
            expect(result).toEqual({ Fixable: 'true' });
        });

        it('returns empty object when all fields are entity scope', () => {
            const result = getSearchFilterWithoutEntityScope({
                Cluster: '"production"',
                Namespace: 'default',
            });
            expect(result).toEqual({});
        });

        it('returns unchanged object when no entity scope fields present', () => {
            const input = { Severity: 'Critical', Fixable: 'true' };
            const result = getSearchFilterWithoutEntityScope(input);
            expect(result).toEqual(input);
        });

        it('does not mutate the input', () => {
            const input = { Cluster: '"production"', Severity: 'Critical' };
            const inputCopy = { ...input };
            getSearchFilterWithoutEntityScope(input);
            expect(input).toEqual(inputCopy);
        });

        it('is case-insensitive for field labels', () => {
            const result = getSearchFilterWithoutEntityScope({
                cluster: '"production"',
                namespace: 'default',
                Severity: 'Critical',
            });
            expect(result).toEqual({ Severity: 'Critical' });
        });
    });

    describe('SearchFilter and EntityScopeRule round-trip', () => {
        it('round-trips a mixed EXACT/REGEX SearchFilter through rules and back', () => {
            const original = {
                Cluster: ['"production"', 'staging.*'],
                'Deployment Label': ['"app=web"'],
                Deployment: ['nginx.*'],
            };

            const rules =
                getEntityScopeRulesFromSearchFilterForClusterNamespaceDeployment(original);
            const result = getSearchFilterFromEntityScopeRules(
                rules,
                searchFieldLabelMapForClusterNamespaceDeployment
            );
            expect(result).toEqual(original);
        });

        it('round-trips entity scope rules through SearchFilter and back', () => {
            const originalRules: EntityScopeRule[] = [
                {
                    entity: 'SCOPE_ENTITY_CLUSTER',
                    field: 'FIELD_NAME',
                    values: [
                        { matchType: 'EXACT', value: 'production' },
                        { matchType: 'REGEX', value: 'staging.*' },
                    ],
                },
                {
                    entity: 'SCOPE_ENTITY_DEPLOYMENT',
                    field: 'FIELD_LABEL',
                    values: [{ matchType: 'EXACT', value: 'app=web' }],
                },
            ];
            const searchFilter = getSearchFilterFromEntityScopeRules(
                originalRules,
                searchFieldLabelMapForClusterNamespaceDeployment
            );
            const result =
                getEntityScopeRulesFromSearchFilterForClusterNamespaceDeployment(searchFilter);
            expect(result).toEqual(originalRules);
        });
    });

    describe('searchValueToRuleValue and ruleValueToSearchValue round-trip', () => {
        it('round-trips an EXACT value', () => {
            const original = '"production"';
            const ruleValue = searchValueToRuleValue(original);
            const result = ruleValueToSearchValue(ruleValue);
            expect(result).toBe(original);
        });

        it('round-trips a REGEX value', () => {
            const original = 'prod.*';
            const ruleValue = searchValueToRuleValue(original);
            const result = ruleValueToSearchValue(ruleValue);
            expect(result).toBe(original);
        });

        it('round-trips a key=value label', () => {
            const original = 'app=frontend';
            const ruleValue = searchValueToRuleValue(original);
            const result = ruleValueToSearchValue(ruleValue);
            expect(result).toBe(original);
        });

        it('round-trips a quoted key=value label', () => {
            const original = '"app=frontend"';
            const ruleValue = searchValueToRuleValue(original);
            const result = ruleValueToSearchValue(ruleValue);
            expect(result).toBe(original);
        });
    });
});
