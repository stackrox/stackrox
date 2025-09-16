import { MitreTechnique } from 'types/mitre.proto';

import { formatTechniqueDisplayName, groupAndSortTechniques } from './MitreTechniqueSelect';

describe('MitreTechniqueSelect', () => {
    describe('formatTechniqueDisplayName', () => {
        test('should remove prefix before colon and space', () => {
            const name = 'T1234: Credential Access';
            const result = formatTechniqueDisplayName(name);
            expect(result).toBe('Credential Access');
        });

        test('should return original name if no colon found', () => {
            const name = 'Simple Technique Name';
            const result = formatTechniqueDisplayName(name);
            expect(result).toBe('Simple Technique Name');
        });

        test('should handle empty string', () => {
            const result = formatTechniqueDisplayName('');
            expect(result).toBe('');
        });

        test('should handle colon without space', () => {
            const name = 'T1234:Credential Access';
            const result = formatTechniqueDisplayName(name);
            expect(result).toBe('Credential Access');
        });
    });

    describe('groupAndSortTechniques', () => {
        const mockTechniques: MitreTechnique[] = [
            {
                id: 'T1234.001',
                name: 'Credential Access: Sub-technique A',
                description: 'Description for sub-technique A',
            },
            {
                id: 'T1234',
                name: 'Credential Access',
                description: 'Description for main technique',
            },
            {
                id: 'T1234.002',
                name: 'Credential Access: Sub-technique B',
                description: 'Description for sub-technique B',
            },
            {
                id: 'T5678',
                name: 'Defense Evasion',
                description: 'Description for single technique',
            },
        ];

        test('should group techniques by base ID', () => {
            const result = groupAndSortTechniques(mockTechniques);

            expect(result).toHaveLength(2);

            const t1234Group = result.find((group) => group.baseId === 'T1234');
            const t5678Group = result.find((group) => group.baseId === 'T5678');

            expect(t1234Group).toBeDefined();
            expect(t5678Group).toBeDefined();

            expect(t1234Group?.techniques).toHaveLength(3);
            expect(t5678Group?.techniques).toHaveLength(1);
        });

        test('should sort base technique first, then sub-techniques alphabetically', () => {
            const result = groupAndSortTechniques(mockTechniques);

            const t1234Group = result.find((group) => group.baseId === 'T1234');
            const techniques = t1234Group?.techniques;

            expect(techniques?.[0].id).toBe('T1234'); // Base technique first
            expect(techniques?.[1].id).toBe('T1234.001'); // Sub-techniques sorted
            expect(techniques?.[2].id).toBe('T1234.002');
        });

        test('should create proper group labels', () => {
            const result = groupAndSortTechniques(mockTechniques);

            const t1234Group = result.find((group) => group.baseId === 'T1234');
            const t5678Group = result.find((group) => group.baseId === 'T5678');

            expect(t1234Group?.groupLabel).toBe('Credential Access');
            expect(t5678Group?.groupLabel).toBe('Defense Evasion');
        });

        test('should handle techniques without dots in ID', () => {
            const simpleTechniques: MitreTechnique[] = [
                {
                    id: 'T1111',
                    name: 'Simple Technique',
                    description: 'Description for simple technique',
                },
            ];

            const result = groupAndSortTechniques(simpleTechniques);

            expect(result).toHaveLength(1);
            expect(result[0].baseId).toBe('T1111');
            expect(result[0].techniques).toHaveLength(1);
            expect(result[0].techniques[0].id).toBe('T1111');
        });

        test('should handle empty array', () => {
            const result = groupAndSortTechniques([]);
            expect(result).toEqual([]);
        });
    });
});
