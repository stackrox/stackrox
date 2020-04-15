const getPaginatedList = (list, currentPage, pageSize) => {
    if (currentPage < 1) {
        throw new Error(
            `Received a page value of ${currentPage}. Only values greater than or equal to 1 are allowed.`
        );
    }
    const startIndex = (currentPage - 1) * pageSize;
    return list.slice(startIndex, startIndex + pageSize);
};

export default getPaginatedList;
