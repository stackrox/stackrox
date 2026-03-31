import { getSearchFilterConfig } from './ViolationsTableSearchFilter.utils';

describe('getSearchFilterConfig', () => {
    it('should return exactly the expected entities for "Applications view"', () => {
        const config = getSearchFilterConfig('Applications view');
        const names = config.map((e) => e.displayName).sort();

        expect(names).toEqual(
            ['Cluster', 'Deployment', 'Namespace', 'Policy', 'Policy violation'].sort()
        );
    });

    it('should return exactly the expected entities for "Platform view"', () => {
        const config = getSearchFilterConfig('Platform view');
        const names = config.map((e) => e.displayName).sort();

        expect(names).toEqual(
            ['Cluster', 'Deployment', 'Namespace', 'Policy', 'Policy violation'].sort()
        );
    });

    it('should return exactly the expected entities for "Node view"', () => {
        const config = getSearchFilterConfig('Node view');
        const names = config.map((e) => e.displayName).sort();

        expect(names).toEqual(['Cluster', 'Node', 'Policy', 'Policy violation'].sort());
    });

    it('should return all entities for "Full view"', () => {
        const config = getSearchFilterConfig('Full view');
        const names = config.map((e) => e.displayName).sort();

        expect(names).toEqual(
            [
                'Cluster',
                'Deployment',
                'Namespace',
                'Node',
                'Policy',
                'Policy violation',
                'Resource',
            ].sort()
        );
    });
});
