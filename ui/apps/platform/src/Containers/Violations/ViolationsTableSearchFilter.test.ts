import { getSearchFilterConfig } from './ViolationsTableSearchFilter';

describe('getSearchFilterConfig', () => {
    it('should exclude Node and Resource entities from "Applications view"', () => {
        const config = getSearchFilterConfig('Applications view');
        const names = config.map((e) => e.displayName);

        expect(names).not.toContain('Node');
        expect(names).not.toContain('Resource');
    });

    it('should exclude Node and Resource entities from "Platform view"', () => {
        const config = getSearchFilterConfig('Platform view');
        const names = config.map((e) => e.displayName);

        expect(names).not.toContain('Node');
        expect(names).not.toContain('Resource');
    });

    it('should exclude Deployment, Namespace, and Resource entities from "Node view"', () => {
        const config = getSearchFilterConfig('Node view');
        const names = config.map((e) => e.displayName);

        expect(names).not.toContain('Deployment');
        expect(names).not.toContain('Namespace');
        expect(names).not.toContain('Resource');
        expect(names).toContain('Node');
    });

    it('should include all entities in "Full view"', () => {
        const config = getSearchFilterConfig('Full view');

        expect(config).toHaveLength(7);
    });
});
