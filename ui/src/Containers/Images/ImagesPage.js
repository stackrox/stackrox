import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import { selectors } from 'reducers';
import { actions as imagesActions, types } from 'reducers/images';

import NoResultsMessage from 'Components/NoResultsMessage';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import ReactRowSelectTable from 'Components/ReactRowSelectTable';
import { sortNumber, sortDate } from 'sorters/sorters';
import ImageDetails from 'Containers/Images/ImageDetails';

class ImagesPage extends Component {
    static propTypes = {
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
        history: ReactRouterPropTypes.history.isRequired,
        location: ReactRouterPropTypes.location.isRequired
    };

    static defaultProps = {
        isFetchingImage: false,
        selectedImage: null
    };

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            this.props.history.push('/main/images');
        }
    };

    updateSelectedImage = image => {
        const urlSuffix = image && image.sha ? `/${image.sha}` : '';
        this.props.history.push({
            pathname: `/main/images${urlSuffix}`,
            search: this.props.location.search
        });
    };

    renderTable() {
        const columns = [
            {
                accessor: 'name',
                Header: 'Image'
            },
            {
                accessor: 'created',
                Header: 'Created at',
                Cell: ({ original }) =>
                    original.created ? dateFns.format(original.created, dateTimeFormat) : '-',
                sortMethod: sortDate
            },
            {
                accessor: 'components',
                Header: 'Components',
                Cell: ({ original }) => original.components || '-',
                sortMethod: sortNumber
            },
            {
                accessor: 'cves',
                Header: 'CVEs',
                Cell: ({ original }) => original.cves || '-',
                sortMethod: sortNumber
            },
            {
                accessor: 'fixableCves',
                Header: 'Fixable',
                Cell: ({ original }) => original.fixableCves || '-',
                sortMethod: sortNumber
            }
        ];
        const { images, selectedImage } = this.props;
        const rows = images;
        const sha = selectedImage && selectedImage.sha;
        if (!rows.length)
            return <NoResultsMessage message="No results found. Please refine your search." />;
        return (
            <ReactRowSelectTable
                rows={rows}
                columns={columns}
                onRowClick={this.updateSelectedImage}
                idAttribute="sha"
                selectedRowId={sha}
                noDataText="No results found. Please refine your search."
            />
        );
    }

    renderSidePanel = () => {
        const { selectedImage } = this.props;
        if (!selectedImage) return '';
        return <ImageDetails image={selectedImage} loading={this.props.isFetchingImage} />;
    };

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 flex-col">
                    <PageHeader header="Images" subHeader={subHeader}>
                        <SearchInput
                            className="flex flex-1"
                            id="images"
                            searchOptions={this.props.searchOptions}
                            searchModifiers={this.props.searchModifiers}
                            searchSuggestions={this.props.searchSuggestions}
                            setSearchOptions={this.props.setSearchOptions}
                            setSearchModifiers={this.props.setSearchModifiers}
                            setSearchSuggestions={this.props.setSearchSuggestions}
                            onSearch={this.onSearch}
                        />
                    </PageHeader>
                    <div className="flex flex-1">
                        <div className="w-full pl-3 pt-3 pr-3 overflow-scroll bg-white rounded-sm bg-base-100">
                            {this.renderTable()}
                        </div>
                        {this.renderSidePanel()}
                    </div>
                </div>
            </section>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getImagesSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const getSelectedImage = (state, props) => {
    const { imageSha } = props.match.params;
    return imageSha ? selectors.getImage(state, imageSha) : null;
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
export default connect(mapStateToProps, mapDispatchToProps)(ImagesPage);
