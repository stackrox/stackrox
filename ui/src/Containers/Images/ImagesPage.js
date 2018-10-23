import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import Table, {
    wrapClassName,
    defaultHeaderClassName,
    defaultColumnClassName
} from 'Components/Table';

import { selectors } from 'reducers';
import { actions as imagesActions, types } from 'reducers/images';

import NoResultsMessage from 'Components/NoResultsMessage';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import { sortNumber, sortDate } from 'sorters/sorters';
import ImageDetails from 'Containers/Images/ImageDetails';
import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';

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

    constructor(props) {
        super(props);
        this.state = {
            page: 0
        };
    }

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            this.props.history.push('/main/images');
        }
    };

    setTablePage = newPage => {
        this.setState({ page: newPage });
    };

    updateSelectedImage = image => {
        const urlSuffix = image && image.id ? `/${image.id}` : '';
        this.props.history.push({
            pathname: `/main/images${urlSuffix}`,
            search: this.props.location.search
        });
    };

    renderPanel = () => {
        const { length } = this.props.images;
        const paginationComponent = (
            <TablePagination
                page={this.state.page}
                dataLength={length}
                setPage={this.setTablePage}
            />
        );
        const headerText = `${length} Image${length === 1 ? '' : 's'} ${
            this.props.isViewFiltered ? 'Matched' : ''
        }`;
        return (
            <Panel header={headerText} headerComponents={paginationComponent}>
                <div className="w-full">{this.renderTable()}</div>
            </Panel>
        );
    };

    renderTable() {
        const columns = [
            {
                accessor: 'name',
                Header: 'Image',
                headerClassName: `w-1/2 ${defaultHeaderClassName}`,
                className: `w-1/2 word-break-all ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ value }) => <span>{value}</span>
            },
            {
                accessor: 'created',
                Header: 'Created at',
                headerClassName: `w-24 ${defaultHeaderClassName}`,
                className: `w-24 ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ original }) =>
                    original.created ? dateFns.format(original.created, dateTimeFormat) : 'N/A',
                sortMethod: sortDate
            },
            {
                accessor: 'components',
                Header: 'Components',
                headerClassName: `w-24 ${defaultHeaderClassName}`,
                className: `w-24 ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ original }) => original.components || 'N/A',
                sortMethod: sortNumber
            },
            {
                accessor: 'cves',
                Header: 'CVEs',
                headerClassName: `w-12 ${defaultHeaderClassName}`,
                className: `w-12 ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ original }) => original.cves || 'N/A',
                sortMethod: sortNumber
            },
            {
                accessor: 'fixableCves',
                Header: 'Fixable',
                headerClassName: `w-16 ${defaultHeaderClassName}`,
                className: `w-16 ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ original }) => original.fixableCves || 'N/A',
                sortMethod: sortNumber
            }
        ];
        const { images, selectedImage } = this.props;
        const rows = images;
        const selectedId = selectedImage && selectedImage.id;
        if (!rows.length)
            return <NoResultsMessage message="No results found. Please refine your search." />;
        return (
            <Table
                rows={rows}
                columns={columns}
                onRowClick={this.updateSelectedImage}
                selectedRowId={selectedId}
                noDataText="No results found. Please refine your search."
                page={this.state.page}
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
            <section className="flex flex-1 flex-col h-full">
                <div className="flex flex-1 flex-col">
                    <PageHeader header="Images" subHeader={subHeader}>
                        <SearchInput
                            className="w-full"
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
                    <div className="flex flex-1 relative">
                        <div className="rounded-sm shadow border-primary-300 bg-base-100 w-full overflow-hidden">
                            {this.renderPanel()}
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
