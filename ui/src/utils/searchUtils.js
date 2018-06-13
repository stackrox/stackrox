/**
 *  Adds a search modifier to the searchOptions
 *
 *  @param {!Object[]} searchOptions an array of search options
 *  @param {!Object[]} modifier a modifier term (ie. 'Cluster:')
 *  @returns {!Object[]} the modified search options
 */
export function addSearchModifier(searchOptions, modifier) {
    const chip = { value: modifier, label: modifier, type: 'categoryOption' };
    return [...searchOptions, chip];
}

/**
 *  Adds a search keyword to the searchOptions
 *
 *  @param {!Object[]} searchOptions an array of search options
 *  @param {!Object[]} keyword a keyword term (ie. 'remote')
 *  @returns {!Object[]} the modified search options
 */
export function addSearchKeyword(searchOptions, keyword) {
    const chip = { value: keyword, label: keyword, className: 'Select-create-option-placeholder' };
    return [...searchOptions, chip];
}

/**
 *  Checks if the modifier exists in the searchOptions
 *
 *  @param {!Object[]} searchOptions an array of search options
 *  @returns {!Object[]} the modified search options
 */
export function hasSearchModifier(searchOptions, modifier) {
    return !!searchOptions.find(
        option => option.type === 'categoryOption' && option.value === modifier
    );
}
