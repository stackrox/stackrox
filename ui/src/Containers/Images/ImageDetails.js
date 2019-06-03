import React, { useState } from 'react';
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

const DockerfileButton = ({ image, openModal }) => {
    const button = (
        <button
            type="button"
            className="flex mx-auto my-2 py-2 px-2 w-5/6 rounded-sm text-primary-700 no-underline hover:bg-primary-200 hover:border-primary-500 uppercase justify-center text-sm items-center bg-base-100 border-2 border-primary-400"
            onClick={openModal}
            disabled={!(image.metadata && image.metadata.v1)}
        >
            View Dockerfile
        </button>
    );
    if (image && image.metadata) return button;
    return (
        <Tooltip placement="top" overlay={<div>Dockerfile not available</div>}>
            <div>{button}</div>
        </Tooltip>
    );
};

DockerfileButton.propTypes = {
    image: PropTypes.shape({
        metadata: PropTypes.shape({})
    }).isRequired,
    openModal: PropTypes.func.isRequired
};

const ImageDetails = ({ image, setDeploymentsSearchOptions, loading, history, location }) => {
    const [modalOpen, setModalOpen] = useState(false);

    function onViewDeploymentsClick() {
        const { name } = image;
        let searchOptions = [];
        searchOptions = addSearchModifier(searchOptions, 'Image:');
        searchOptions = addSearchKeyword(searchOptions, name);
        setDeploymentsSearchOptions(searchOptions);
        history.push('/main/risk');
    }

    function openModal() {
        setModalOpen(true);
    }

    function closeModal() {
        setModalOpen(false);
    }

    function unselectImage() {
        history.push({
            pathname: `/main/images`,
            search: location.search
        });
    }

    function renderOverview() {
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
                    <CollapsibleCard title="Overview">
                        <div className="h-full">
                            <div className="px-3">
                                <KeyValuePairs data={imageDetail} keyValueMap={imageDetailsMap} />
                            </div>
                            <div className="flex bg-primary-100">
                                <span className="w-1/2">
                                    <button
                                        type="button"
                                        className="flex mx-auto my-2 py-2 px-2 w-5/6 rounded-sm text-primary-700 no-underline hover:bg-primary-200 hover:border-primary-500 uppercase justify-center text-sm items-center bg-base-100 border-2 border-primary-400"
                                        onClick={onViewDeploymentsClick}
                                    >
                                        View Deployments
                                    </button>
                                </span>
                                <span className="w-1/2 border-base-300 border-l-2">
                                    <DockerfileButton image={image} openModal={openModal} />
                                </span>
                            </div>
                        </div>
                    </CollapsibleCard>
                </div>
            </div>
        );
    }

    if (!image) return '';
    const { scan, fixableCves, name: header } = image;
    const content = loading ? (
        <Loader />
    ) : (
        <div className="flex flex-col w-full bg-base-200 overflow-auto pb-5">
            {renderOverview()}
            <div className="px-3 pt-5">
                <div className="alert-preview bg-base-100 shadow text-primary-600">
                    <CollapsibleCard title="CVEs">
                        <div className="h-full">
                            <CVETable scan={scan} containsFixableCVEs={fixableCves > 0} />
                        </div>
                    </CollapsibleCard>
                </div>
            </div>
            <DockerfileModal modalOpen={modalOpen} image={image} onClose={closeModal} />
        </div>
    );
    return (
        <Panel
            header={header}
            onClose={unselectImage}
            className="bg-primary-200 z-10 w-full h-full absolute pin-r pin-t md:w-1/2 min-w-72 md:relative"
        >
            {content}
        </Panel>
    );
};

ImageDetails.propTypes = {
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

const mapDispatchToProps = {
    setDeploymentsSearchOptions: deploymentsActions.setDeploymentsSearchOptions
};

export default withRouter(
    connect(
        null,
        mapDispatchToProps
    )(ImageDetails)
);
