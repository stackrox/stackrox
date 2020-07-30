import getPaginatedList from './getPaginatedList';

describe('getPaginatedList', () => {
    it('should return the first 5 items of the list', () => {
        const list = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
        const expectedList = [1, 2, 3, 4, 5];
        const currentPage = 1;
        const pageSize = 5;

        const paginatedList = getPaginatedList(list, currentPage, pageSize);

        expect(paginatedList).toEqual(expectedList);
    });

    it('should return the last 5 items of the list', () => {
        const list = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
        const expectedList = [6, 7, 8, 9, 10];
        const currentPage = 2;
        const pageSize = 5;

        const paginatedList = getPaginatedList(list, currentPage, pageSize);

        expect(paginatedList).toEqual(expectedList);
    });

    it('should not paginate with a page value less than 1', () => {
        const list = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
        const expectedError = new Error(
            'Received a page value of 0. Only values greater than or equal to 1 are allowed.'
        );
        const currentPage = 0;
        const pageSize = 5;

        // In Jest you have to pass a function into expect(function).toThrow(blank or type of error).
        const paginateWithZeroIndex = () => {
            getPaginatedList(list, currentPage, pageSize);
        };

        expect(paginateWithZeroIndex).toThrow(expectedError);
    });
});
