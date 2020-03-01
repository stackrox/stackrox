import React, { useState } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import dateFns from 'date-fns';
import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';

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
    id: {
        label: 'Digest',
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
    const disabled = !image.metadata?.v1;
    const button = (
        <button
            type="button"
            className="flex mx-auto my-2 py-2 px-2 w-5/6 rounded-sm text-primary-700 no-underline hover:bg-primary-200 hover:border-primary-500 uppercase justify-center text-sm items-center bg-base-100 border-2 border-primary-400"
            onClick={openModal}
            disabled={disabled}
        >
            View Dockerfile
        </button>
    );
    if (!disabled) return button;
    return (
        <Tooltip content={<TooltipOverlay>Dockerfile not available</TooltipOverlay>}>
            <div>{button}</div>
        </Tooltip>
    );
};

DockerfileButton.propTypes = {
    image: PropTypes.shape({
        metadata: PropTypes.shape({
            v1: PropTypes.any
        })
    }).isRequired,
    openModal: PropTypes.func.isRequired
};

const ImageDetails = ({
    image,
    setSelectedImageId,
    setDeploymentsSearchOptions,
    loading,
    history
}) => {
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
        setSelectedImageId(undefined);
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
        <div className="h-full flex flex-col w-full bg-base-200 overflow-auto pb-5">
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
            className="bg-primary-200 w-full h-full absolute right-0 top-0 md:w-2/3 min-w-1/2 md:relative"
        >
            {content}
        </Panel>
    );
};

ImageDetails.propTypes = {
    image: PropTypes.shape({
        name: PropTypes.string.isRequired,
        scan: PropTypes.shape({
            scanTime: PropTypes.string
        }),
        fixableCves: PropTypes.number,
        metadata: PropTypes.shape({
            v1: PropTypes.any
        }),
        id: PropTypes.string,
        components: PropTypes.any,
        cves: PropTypes.any
    }).isRequired,
    loading: PropTypes.bool.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    setSelectedImageId: PropTypes.func.isRequired,
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
