import React, { useState } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as imagesActions, types } from 'reducers/images';

import PageHeader, { PageHeaderComponent } from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import ImageDetails from 'Containers/Images/ImageDetails';
import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import ImagesTable from './ImagesTable';

const ImageSidePanel = ({ selectedImage, isFetchingImage }) => {
    if (!selectedImage) return '';
    return <ImageDetails image={selectedImage} loading={isFetchingImage} />;
};

ImageSidePanel.propTypes = {
    selectedImage: PropTypes.shape({}),
    isFetchingImage: PropTypes.bool
};

ImageSidePanel.defaultProps = {
    isFetchingImage: false,
    selectedImage: null
};

const ImagesPage = ({
    history,
    images,
    isViewFiltered,
    selectedImage,
    searchModifiers,
    searchOptions,
    searchSuggestions,
    setSearchOptions,
    setSearchModifiers,
    setSearchSuggestions,
    isFetchingImage
}) => {
    const [page, setPage] = useState(0);
    function onSearch(newSearchOptions) {
        if (newSearchOptions.length && !newSearchOptions[newSearchOptions.length - 1].type) {
            history.push('/main/images');
        }
    }

    const subHeader = isViewFiltered ? 'Filtered view' : 'Default view';
    const defaultOption = searchModifiers.find(x => x.value === 'Image:');
    const { length } = images;
    const paginationComponent = (
        <TablePagination page={page} dataLength={length} setPage={setPage} />
    );

    const headerComponent = (
        <PageHeaderComponent length={length} type="Image" isViewFiltered={isViewFiltered} />
    );
    return (
        <section className="flex flex-1 flex-col h-full">
            <div className="flex flex-1 flex-col">
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
                        onSearch={onSearch}
                        defaultOption={defaultOption}
                        autoCompleteCategories={['IMAGES']}
                    />
                </PageHeader>
                <div className="flex flex-1 relative">
                    <div className="shadow border-primary-300 bg-base-100 w-full overflow-hidden">
                        <Panel
                            headerTextComponent={headerComponent}
                            headerComponents={paginationComponent}
                        >
                            <div className="w-full">
                                <ImagesTable
                                    rows={images}
                                    selectedImage={selectedImage}
                                    page={page}
                                />
                            </div>
                        </Panel>
                    </div>
                    <ImageSidePanel
                        isFetchingImage={isFetchingImage}
                        selectedImage={selectedImage}
                    />
                </div>
            </div>
        </section>
    );
};

ImagesPage.propTypes = {
    images: PropTypes.arrayOf(PropTypes.object).isRequired,
    selectedImage: PropTypes.shape({}),
    searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
    setSearchOptions: PropTypes.func.isRequired,
    setSearchModifiers: PropTypes.func.isRequired,
    setSearchSuggestions: PropTypes.func.isRequired,
    isViewFiltered: PropTypes.bool.isRequired,
    isFetchingImage: PropTypes.bool,
    history: ReactRouterPropTypes.history.isRequired
};

ImagesPage.defaultProps = {
    isFetchingImage: false,
    selectedImage: null
};

const isViewFiltered = createSelector(
    [selectors.getImagesSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const getSelectedImage = (state, props) => {
    const { imageId } = props.match.params;
    return imageId ? selectors.getImage(state, imageId) : null;
};

const mapStateToProps = createStructuredSelector({
    images: selectors.getFilteredImages,
    selectedImage: getSelectedImage,
    searchOptions: selectors.getImagesSearchOptions,
    searchModifiers: selectors.getImagesSearchModifiers,
    searchSuggestions: selectors.getImagesSearchSuggestions,
    isViewFiltered,
    isFetchingImage: state => selectors.getLoadingStatus(state, types.FETCH_IMAGE)
});

const mapDispatchToProps = {
    setSearchOptions: imagesActions.setImagesSearchOptions,
    setSearchModifiers: imagesActions.setImagesSearchModifiers,
    setSearchSuggestions: imagesActions.setImagesSearchSuggestions,
    fetchImage: imagesActions.fetchImage
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(ImagesPage);
