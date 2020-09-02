import { getLastCategoryInSearchOptions, createSearchModifiers } from './SearchInput';

describe('SearchInput', () => {
    describe('getLastCategoryInSearchOptions', () => {
        it('should get no category when theres no category in the search options', () => {
            const searchOptions = [{ label: 'Apple', value: 'Apple' }];
            const category = getLastCategoryInSearchOptions(searchOptions);
            expect(category).toEqual(null);
        });

        it('should get the last category when theres only one category', () => {
            const searchOptions = [
                { label: 'Fruit:', value: 'Fruit:', type: 'categoryOption' },
                { label: 'Apple', value: 'Apple' },
                { label: 'Banana', value: 'Banana' },
            ];
            const category = getLastCategoryInSearchOptions(searchOptions);
            expect(category).toEqual('Fruit');
        });

        it('should get the last category when theres more than one category', () => {
            const searchOptions = [
                { label: 'Fruit:', value: 'Fruit:', type: 'categoryOption' },
                { label: 'Apple', value: 'Apple' },
                { label: 'Banana', value: 'Banana' },
                { label: 'Superhero:', value: 'Superhero:', type: 'categoryOption' },
                { label: 'Batman', value: 'Batman' },
                { label: 'Superman', value: 'Superman' },
            ];
            const category = getLastCategoryInSearchOptions(searchOptions);
            expect(category).toEqual('Superhero');
        });
    });

    describe('createSearchModifiers', () => {
        it('should create search modifiers based on an array of category strings', () => {
            const categories = ['Fruit', 'Superhero'];
            const searchModifiers = createSearchModifiers(categories);
            expect(searchModifiers).toEqual([
                { label: 'Fruit:', value: 'Fruit:', type: 'categoryOption' },
                { label: 'Superhero:', value: 'Superhero:', type: 'categoryOption' },
            ]);
        });
    });
});
