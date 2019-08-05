import React, { useEffect } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as imagesActions } from 'reducers/images';
import { fetchImages, fetchImageCount } from 'services/ImagesService';

import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import { pageSize } from 'Components/Table';

function ImagesPageHeader({
    currentPage,
    setCurrentImages,
    setImagesCount,
    setSelectedImageId,
    isViewFiltered,
    setIsViewFiltered,
    sortOption,
    searchOptions,
    searchModifiers,
    searchSuggestions,
    setSearchOptions,
    setSearchModifiers,
    setSearchSuggestions
}) {
    const hasExecutableFilter =
        searchOptions.length && !searchOptions[searchOptions.length - 1].type;
    const hasNoFilter = !searchOptions.length;

    if (hasExecutableFilter && !isViewFiltered) {
        setIsViewFiltered(true);
    } else if (hasNoFilter && isViewFiltered) {
        setIsViewFiltered(false);
    }
    if (hasExecutableFilter) {
        setSelectedImageId(undefined);
    }

    useEffect(
        () => {
            if (!searchOptions.length || !searchOptions[searchOptions.length - 1].type) {
                fetchImages(searchOptions, sortOption, currentPage, pageSize).then(images =>
                    setCurrentImages(images)
                );
                fetchImageCount(searchOptions).then(count => setImagesCount(count));
            }
        },
        [searchOptions, sortOption, currentPage, setCurrentImages, setImagesCount]
    );

    const subHeader = isViewFiltered ? 'Filtered view' : 'Default view';
    const defaultOption = searchModifiers.find(x => x.value === 'Image:');
    return (
        <PageHeader header="Images" subHeader={subHeader}>
            <SearchInput
                className="w-full"
                id="images"
                searchOptions={searchOptions}
                searchModifiers={searchModifiers}
                searchSuggestions={searchSuggestions}
                setSearchOptions={setSearchOptions}
                setSearchModifiers={setSearchModifiers}
                setSearchSuggestions={setSearchSuggestions}
                defaultOption={defaultOption}
                autoCompleteCategories={['IMAGES']}
            />
        </PageHeader>
    );
}

ImagesPageHeader.propTypes = {
    currentPage: PropTypes.number.isRequired,
    setCurrentImages: PropTypes.func.isRequired,
    setImagesCount: PropTypes.func.isRequired,
    setSelectedImageId: PropTypes.func.isRequired,
    isViewFiltered: PropTypes.bool.isRequired,
    setIsViewFiltered: PropTypes.func.isRequired,
    sortOption: PropTypes.shape({}).isRequired,

    // Search specific input.
    searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
    setSearchOptions: PropTypes.func.isRequired,
    setSearchModifiers: PropTypes.func.isRequired,
    setSearchSuggestions: PropTypes.func.isRequired
};

// Still have to use redux for search.
const mapStateToProps = createStructuredSelector({
    searchOptions: selectors.getImagesSearchOptions,
    searchModifiers: selectors.getImagesSearchModifiers,
    searchSuggestions: selectors.getImagesSearchSuggestions
});

const mapDispatchToProps = {
    setSearchOptions: imagesActions.setImagesSearchOptions,
    setSearchModifiers: imagesActions.setImagesSearchModifiers,
    setSearchSuggestions: imagesActions.setImagesSearchSuggestions
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(ImagesPageHeader);
