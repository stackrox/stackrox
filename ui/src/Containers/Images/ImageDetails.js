import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import dateFns from 'date-fns';
import Tooltip from 'rc-tooltip';

import dateTimeFormat from 'constants/dateTimeFormat';

import { actions as deploymentsActions } from 'reducers/deployments';
import { addSearchModifier, addSearchKeyword } from 'utils/searchUtils';

import CollapsibleCard from 'Components/CollapsibleCard';
import Panel from 'Components/Panel';
import KeyValuePairs from 'Components/KeyValuePairs';
import DockerfileModal from 'Containers/Images/DockerfileModal';
import Loader from 'Components/Loader';
import CVETable from './CVETable';

const imageDetailsMap = {
    scanTime: {
        label: 'Last scan time',
        formatValue: timestamp =>
            timestamp ? dateFns.format(timestamp, dateTimeFormat) : 'not available'
    },
    sha: {
        label: 'SHA',
        className: 'word-break'
    },
    totalComponents: {
        label: 'Components'
    },
    totalCVEs: {
        label: 'CVEs'
    }
};

class ImageDetails extends Component {
    static propTypes = {
        image: PropTypes.shape({
            name: PropTypes.string.isRequired,
            scan: PropTypes.shape({}),
            fixableCves: PropTypes.number
        }).isRequired,
        loading: PropTypes.bool.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
        location: ReactRouterPropTypes.location.isRequired,
        setDeploymentsSearchOptions: PropTypes.func.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            modalOpen: false
        };
    }

    onViewDeploymentsClick = () => {
        const { name } = this.props.image;
        let searchOptions = [];
        searchOptions = addSearchModifier(searchOptions, 'Image:');
        searchOptions = addSearchKeyword(searchOptions, name);
        this.props.setDeploymentsSearchOptions(searchOptions);
        this.props.history.push('/main/risk');
    };

    getDockerFileButton = image => {
        const button = (
            <button
                type="button"
                className="flex mx-auto my-2 py-2 px-2 w-5/6 rounded-sm text-primary-700 no-underline hover:bg-primary-200 hover:border-primary-500 uppercase justify-center text-sm items-center bg-base-100 border-2 border-primary-400"
                onClick={this.openModal}
                disabled={!(image.metadata && image.metadata.v1)}
            >
                View Dockerfile
            </button>
        );
        if (image.metadata) return button;
        return (
            <Tooltip placement="top" overlay={<div>Dockerfile not available</div>}>
                <div>{button}</div>
            </Tooltip>
        );
    };

    openModal = () => {
        this.setState({ modalOpen: true });
    };

    closeModal = () => {
        this.setState({ modalOpen: false });
    };

    updateSelectedImage = image => {
        const urlSuffix = image && image.id ? `/${image.id}` : '';
        this.props.history.push({
            pathname: `/main/images${urlSuffix}`,
            search: this.props.location.search
        });
    };

    renderOverview = () => {
        const title = 'Overview';
        const { image } = this.props;
        if (!image) return null;
        const imageDetail = {
            scanTime: image.scan ? image.scan.scanTime : '',
            id: image.id,
            totalComponents: image.components ? image.components : 'not available',
            totalCVEs: image.cves ? image.cves : []
        };
        return (
            <div className="px-3 pt-5">
                <div className="alert-preview bg-base-100 shadow text-primary-600">
                    <CollapsibleCard title={title}>
                        <div className="h-full">
                            <div className="px-3">
                                <KeyValuePairs data={imageDetail} keyValueMap={imageDetailsMap} />
                            </div>
                            <div className="flex bg-primary-100">
                                <span className="w-1/2">
                                    <button
                                        type="button"
                                        className="flex mx-auto my-2 py-2 px-2 w-5/6 rounded-sm text-primary-700 no-underline hover:bg-primary-200 hover:border-primary-500 uppercase justify-center text-sm items-center bg-base-100 border-2 border-primary-400"
                                        onClick={this.onViewDeploymentsClick}
                                    >
                                        View Deployments
                                    </button>
                                </span>
                                <span className="w-1/2 border-base-300 border-l-2">
                                    {this.getDockerFileButton(image)}
                                </span>
                            </div>
                        </div>
                    </CollapsibleCard>
                </div>
            </div>
        );
    };

    renderCVEs = () => {
        const title = 'CVEs';
        return (
            <div className="px-3 pt-5">
                <div className="alert-preview bg-base-100 shadow text-primary-600">
                    <CollapsibleCard title={title}>
                        <div className="h-full"> {this.renderCVEsTable()}</div>
                    </CollapsibleCard>
                </div>
            </div>
        );
    };

    renderCVEsTable = () => {
        const { scan, fixableCves } = this.props.image;
        if (!scan) return <div className="p-3">No scanner setup for this registry</div>;
        return <CVETable components={scan.components} isFixable={fixableCves} />;
    };

    renderDockerfileModal() {
        const { image } = this.props;
        if (!this.state.modalOpen || !image || !image.metadata || !image.metadata.v1) return null;
        return <DockerfileModal image={image} onClose={this.closeModal} />;
    }

    render() {
        const { image, loading } = this.props;
        if (!image) return '';
        const header = image.name;
        const content = loading ? (
            <Loader />
        ) : (
            <div className="flex flex-col w-full bg-base-200 overflow-auto pb-5">
                {this.renderOverview()}
                {this.renderCVEs()}
                {this.renderDockerfileModal()}
            </div>
        );
        return (
            <Panel
                header={header}
                onClose={this.updateSelectedImage}
                className="bg-primary-200 z-10 w-full h-full absolute pin-r pin-t md:w-1/2 min-w-72 md:relative"
            >
                {content}
            </Panel>
        );
    }
}

const mapDispatchToProps = {
    setDeploymentsSearchOptions: deploymentsActions.setDeploymentsSearchOptions
};

export default withRouter(
    connect(
        null,
        mapDispatchToProps
    )(ImageDetails)
);
