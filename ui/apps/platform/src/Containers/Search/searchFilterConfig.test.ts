import { searchFilterConfig } from './searchFilterConfig';

describe('searchFilterConfig', () => {
    it('includes one entity per global search result category', () => {
        expect(searchFilterConfig).toHaveLength(13);
    });

    it('has a unique display name for every entity', () => {
        const displayNames = searchFilterConfig.map((entity) => entity.displayName);
        expect(new Set(displayNames).size).toBe(displayNames.length);
    });

    it('has a unique search category for every entity', () => {
        const searchCategories = searchFilterConfig.map((entity) => entity.searchCategory);
        expect(new Set(searchCategories).size).toBe(searchCategories.length);
    });

    it('gives every entity at least one attribute', () => {
        searchFilterConfig.forEach((entity) => {
            expect(entity.attributes.length).toBeGreaterThan(0);
        });
    });

    // Regression guard: CompoundSearchFilterLabels renders the chip row by flattening the
    // attributes of every entity, so a search term shared by two entities renders a duplicate
    // chip (this is what caused "Violation time" to appear twice).
    it('does not reuse a search term across entities', () => {
        const searchTerms = searchFilterConfig.flatMap((entity) =>
            entity.attributes.map((attribute) => attribute.searchTerm)
        );
        const duplicates = searchTerms.filter((term, index) => searchTerms.indexOf(term) !== index);
        expect(duplicates).toEqual([]);
    });

    it('exposes the Violation entity with a human-readable Time attribute', () => {
        const violation = searchFilterConfig.find((entity) => entity.displayName === 'Violation');
        expect(violation).toBeDefined();

        const timeAttribute = violation?.attributes.find(
            (attribute) => attribute.searchTerm === 'Violation Time'
        );
        expect(timeAttribute?.displayName).toBe('Time');
    });

    it('includes a Deployment entity to serve as the default', () => {
        expect(searchFilterConfig.some((entity) => entity.displayName === 'Deployment')).toBe(true);
    });
});
