import type { ClientPolicy, ClientPolicyGroup } from 'types/policy.proto';
import { getAvailableOptionsForField } from './policyCriteriaUtils';

describe('policyCriteriaUtils', () => {
    describe('getAvailableOptionsForField', () => {
        const mockOptions = [
            { label: 'Option A', value: 'A' },
            { label: 'Option B', value: 'B' },
            { label: 'Option C', value: 'C' },
        ];

        const mockFieldWithValues = (values: string[]): ClientPolicyGroup => {
            return {
                fieldName: 'Port Exposure Method',
                booleanOperator: 'OR',
                negate: false,
                values: values.map((value) => ({ value })),
            };
        };

        test('returns empty array when options array is empty', () => {
            const mockPolicy = {
                policySections: [],
            };

            const result = getAvailableOptionsForField(
                [],
                'policySections[0].policyGroups[0].values[0]',
                mockPolicy
            );

            expect(result).toEqual([]);
        });

        test('returns all options when no values are selected', () => {
            const mockPolicy = {
                policySections: [
                    {
                        sectionName: 'Rule 1',
                        policyGroups: [mockFieldWithValues([''])],
                    },
                ],
            } satisfies Pick<ClientPolicy, 'policySections'>;

            const result = getAvailableOptionsForField(
                mockOptions,
                'policySections[0].policyGroups[0].values[0]',
                mockPolicy
            );

            expect(result).toEqual(mockOptions);
        });

        test('filters out already-selected options from other values in the same group', () => {
            const mockPolicy = {
                policySections: [
                    {
                        sectionName: 'Rule 1',
                        policyGroups: [mockFieldWithValues(['A', 'B', ''])],
                    },
                ],
            } satisfies Pick<ClientPolicy, 'policySections'>;

            // For the third value (empty), should filter out A and B
            const result = getAvailableOptionsForField(
                mockOptions,
                'policySections[0].policyGroups[0].values[2]',
                mockPolicy
            );

            expect(result).toEqual([{ label: 'Option C', value: 'C' }]);
        });

        test('does not filter out the current field value', () => {
            const mockPolicy = {
                policySections: [
                    {
                        sectionName: 'Rule 1',
                        policyGroups: [mockFieldWithValues(['A', 'B'])],
                    },
                ],
            } satisfies Pick<ClientPolicy, 'policySections'>;

            // For the first value (A), should still include A in options
            const result = getAvailableOptionsForField(
                mockOptions,
                'policySections[0].policyGroups[0].values[0]',
                mockPolicy
            );

            expect(result).toEqual([
                { label: 'Option A', value: 'A' },
                { label: 'Option C', value: 'C' },
            ]);
        });

        test('returns empty array when all options are selected', () => {
            const mockPolicy = {
                policySections: [
                    {
                        sectionName: 'Rule 1',
                        policyGroups: [mockFieldWithValues(['A', 'B', 'C', ''])],
                    },
                ],
            } satisfies Pick<ClientPolicy, 'policySections'>;

            // For the fourth value (empty), all options are taken
            const result = getAvailableOptionsForField(
                mockOptions,
                'policySections[0].policyGroups[0].values[3]',
                mockPolicy
            );

            expect(result).toEqual([]);
        });

        test('only filters within the same policy group', () => {
            const mockPolicy = {
                policySections: [
                    {
                        sectionName: 'Rule 1',
                        policyGroups: [mockFieldWithValues(['A', '']), mockFieldWithValues([''])],
                    },
                ],
            } satisfies Pick<ClientPolicy, 'policySections'>;

            // Second group should have all options available (A is in different group)
            const resultGroup1 = getAvailableOptionsForField(
                mockOptions,
                'policySections[0].policyGroups[0].values[1]',
                mockPolicy
            );

            expect(resultGroup1).toEqual([
                { label: 'Option B', value: 'B' },
                { label: 'Option C', value: 'C' },
            ]);

            const resultGroup2 = getAvailableOptionsForField(
                mockOptions,
                'policySections[0].policyGroups[1].values[0]',
                mockPolicy
            );

            expect(resultGroup2).toEqual(mockOptions);
        });

        test('returns all options for invalid group path', () => {
            const mockPolicy = {
                policySections: [],
            };

            const result = getAvailableOptionsForField(
                mockOptions,
                'invalid-field-name',
                mockPolicy
            );

            expect(result).toEqual(mockOptions);
        });
    });
});
