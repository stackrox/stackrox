import { renderHook } from '@testing-library/react-hooks';

import useAutoCompleteResults from './useAutoCompleteResults';

describe('useAutoCompleteResults', () => {
    it('should show autocomplete results for the data', () => {
        const data = [
            { player: 'Bob', wins: 1, losses: 0 },
            { player: 'Bill', wins: 2, losses: 5 },
            { player: 'Alice', wins: 3, losses: 10 },
            { player: 'Jill', wins: 4, losses: 11 },
            { player: 'Jacob', wins: 5, losses: 50 },
        ];
        const searchOptions = [
            { label: 'Wins:', value: 'Wins:', type: 'categoryOption' },
            { label: '1', value: '1' },
            { label: 'Losses:', value: 'Losses:', type: 'categoryOption' },
        ];
        const categories = ['Wins', 'Losses'];
        const getDataValueByCategory = (datum, category) => {
            const dataValueByCategory = {
                Wins: (d) => d.wins.toString(),
                Losses: (d) => d.losses.toString(),
            };
            return dataValueByCategory[category](datum);
        };
        const { result } = renderHook(() =>
            useAutoCompleteResults(data, searchOptions, categories, getDataValueByCategory)
        );
        expect(result.current).toEqual(['0', '5', '10', '11', '50']);
    });
});
