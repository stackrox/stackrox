import React, { useEffect, useState } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';

import RiskPageHeader from './RiskPageHeader';
import RiskSidePanel from './RiskSidePanel';
import RiskTablePanel from './RiskTablePanel';

const RiskPage = ({
    history,
    location: { search },
    match: {
        params: { deploymentId }
    }
}) => {
    // Handle changes to applied search options.
    const [isViewFiltered, setIsViewFiltered] = useState(false);

    // Handle changes to the currently selected deployment.
    const [selectedDeploymentId, setSelectedDeploymentId] = useState(deploymentId);

    // Page changes.
    const [currentPage, setCurrentPage] = useState(0);

    // The currently loaded deployments, and the sort option.
    const [currentDeployments, setCurrentDeployments] = useState([]);
    const [sortOption, setSortOption] = useState({ field: 'Priority', reversed: false });

    // The current number of deployments that match the query.
    const [deploymentsCount, setDeploymentsCount] = useState(0);

    // When the selected deployment changes, update the URL.
    useEffect(
        () => {
            const urlSuffix = selectedDeploymentId ? `/${selectedDeploymentId}` : '';
            history.push({
                pathname: `/main/risk${urlSuffix}`,
                search
            });
        },
        [selectedDeploymentId, history, search]
    );

    return (
        <section className="flex flex-1 flex-col h-full">
            <div className="flex flex-1 flex-col">
                <RiskPageHeader
                    currentPage={currentPage}
                    setCurrentDeployments={setCurrentDeployments}
                    setDeploymentsCount={setDeploymentsCount}
                    setSelectedDeploymentId={setSelectedDeploymentId}
                    isViewFiltered={isViewFiltered}
                    setIsViewFiltered={setIsViewFiltered}
                    sortOption={sortOption}
                />
                <div className="flex flex-1 relative">
                    <div className="shadow border-primary-300 bg-base-100 w-full overflow-hidden">
                        <RiskTablePanel
                            currentDeployments={currentDeployments}
                            currentPage={currentPage}
                            setCurrentPage={setCurrentPage}
                            deploymentCount={deploymentsCount}
                            selectedDeploymentId={selectedDeploymentId}
                            setSelectedDeploymentId={setSelectedDeploymentId}
                            setSortOption={setSortOption}
                            isViewFiltered={isViewFiltered}
                        />
                    </div>
                    <RiskSidePanel
                        selectedDeploymentId={selectedDeploymentId}
                        setSelectedDeploymentId={setSelectedDeploymentId}
                    />
                </div>
            </div>
        </section>
    );
};

RiskPage.propTypes = {
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    match: ReactRouterPropTypes.match.isRequired
};

export default RiskPage;
