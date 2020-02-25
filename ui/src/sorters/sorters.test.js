import { sortAscii } from './sorters';

describe('sorters', () => {
    describe('sortAscii', () => {
        it('should sort a list of strings alphabetically, respecting ASCII casing', () => {
            const unsortedArr = [
                'Process Targeting Kubernetes Service Endpoint',
                'crontab Execution',
                'Shellshock: Multiple CVEs',
                '90-Day Image Age',
                'Latest tag',
                'CAP_SYS_ADMIN capability added'
            ];

            const sortedArr = unsortedArr.sort(sortAscii);

            expect(sortedArr).toEqual([
                '90-Day Image Age',
                'CAP_SYS_ADMIN capability added',
                'Latest tag',
                'Process Targeting Kubernetes Service Endpoint',
                'Shellshock: Multiple CVEs',
                'crontab Execution'
            ]);
        });
    });
});
