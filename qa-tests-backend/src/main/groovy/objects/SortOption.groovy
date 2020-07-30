package objects

class SortOption {
    String  field
    Boolean reversed

    SortOption(String field) {
        this.field = field
    }

    SortOption(String field, Boolean reversed) {
        this.field = field
        this.reversed = reversed
    }

}
