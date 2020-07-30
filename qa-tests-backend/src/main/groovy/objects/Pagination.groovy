package objects

class Pagination {
    int limit
    int offset
    SortOption sortOption

    Pagination(int limit) {
        this.limit = limit
    }

    Pagination(int limit, int offset) {
        this.limit = limit
        this.offset = offset
    }

    Pagination(int limit, int offset, SortOption sortOption) {
        this.limit = limit
        this.offset = offset
        this.sortOption = sortOption
    }

}
