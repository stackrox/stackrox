import { renderHook } from '@testing-library/react-hooks';

import useSearchFilteredData, { getCategoryValuesPairs } from './useSearchFilteredData';

describe('useSearchFilteredData', () => {
    describe('getCategoryValuesPairs', () => {
        it('should return category/value pairs', () => {
            const searchOptions = [
                { label: 'Fruit:', value: 'Fruit:', type: 'categoryOption' },
                { label: 'Apple', value: 'Apple' },
                { label: 'Banana', value: 'Banana' },
                { label: 'Superhero:', value: 'Superhero:', type: 'categoryOption' },
                { label: 'Batman', value: 'Batman' },
                { label: 'Superman', value: 'Superman' },
            ];
            const { result } = renderHook(() => getCategoryValuesPairs(searchOptions));
            expect(result.current).toEqual([
                { category: 'Fruit', values: ['Apple', 'Banana'] },
                { category: 'Superhero', values: ['Batman', 'Superman'] },
            ]);
        });
    });

    describe('useSearchFilteredData', () => {
        it('should filter data using search options', () => {
            const data = [
                { player: 'Bob', wins: 1 },
                { player: 'Bill', wins: 2 },
                { player: 'Alice', wins: 3 },
                { player: 'Jill', wins: 4 },
                { player: 'Jacob', wins: 5 },
            ];
            const searchOptions = [
                { label: 'Wins:', value: 'Wins:', type: 'categoryOption' },
                { label: '1', value: '1' },
                { label: '5', value: '5' },
            ];
            const getDataValueByCategory = (datum, category) => {
                const categoryToValuesMap = {
                    Wins: (d) => d.wins.toString(),
                };
                return categoryToValuesMap[category](datum);
            };
            const { result } = renderHook(() =>
                useSearchFilteredData(data, searchOptions, getDataValueByCategory)
            );
            expect(result.current).toEqual([
                { player: 'Bob', wins: 1 },
                { player: 'Jacob', wins: 5 },
            ]);
        });
    });
});
