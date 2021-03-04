import React, { useEffect, useState } from 'react';
import { Message } from '@stackrox/ui-components';

import PageHeader from 'Components/PageHeader';
import Widget from 'Components/Widget';
import ViewAllButton from 'Components/ViewAllButton';
import useInterval from 'hooks/useInterval';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import { clustersBasePath } from 'routePaths';
import { fetchClustersAsArray } from 'services/ClustersService';
import { fetchVulnerabilityDefinitionsInfo } from 'services/IntegrationHealthService';

import ClusterOverview from './Components/ClusterOverview';
import CollectorStatus from './Components/CollectorStatus';
import AdmissionControlStatus from './Components/AdmissionControlStatus';
import CredentialExpiration from './Components/CredentialExpiration';
import GenerateDiagnosticBundleButton from './Components/GenerateDiagnosticBundleButton';
import SensorStatus from './Components/SensorStatus';
import SensorUpgrade from './Components/SensorUpgrade';
import VulnerabilityDefinitions from './Components/VulnerabilityDefinitions';
import ImageIntegrationHealthWidget from './Components/ImageIntegrationHealthWidget';
import NotifierIntegrationHealthWidget from './Components/NotifierIntegrationHealthWidget';
import BackupIntegrationHealthWidget from './Components/BackupIntegrationHealthWidget';
import LogIntegrationHealthWidget from './Components/LogIntegrationHealthWidget';

const SystemHealthDashboardPage = () => {
    const isK8sAuditLoggingEnabled = useFeatureFlagEnabled(
        knownBackendFlags.ROX_K8S_AUDIT_LOG_DETECTION
    );
    const [pollingCountFaster, setPollingCountFaster] = useState(0);
    const [pollingCountSlower, setPollingCountSlower] = useState(0);
    const [currentDatetime, setCurrentDatetime] = useState(null);

    const [clusters, setClusters] = useState([]);
    const [vulnerabilityDefinitionsInfo, setVulnerabilityDefinitionsInfo] = useState(null);

    const [clustersRequestHasError, setClustersRequestHasError] = useState(false);
    const [
        vulnerabilityDefinitionsRequestHasError,
        setVulnerabilityDefinitionsRequestHasError,
    ] = useState(false);

    useEffect(() => {
        setCurrentDatetime(new Date());
        fetchClustersAsArray()
            .then((array) => {
                setClusters(array);
                setClustersRequestHasError(false);
            })
            .catch(() => {
                setClusters([]);
                setClustersRequestHasError(true);
            });
    }, [pollingCountFaster]);

    useEffect(() => {
        fetchVulnerabilityDefinitionsInfo()
            .then((info) => {
                setVulnerabilityDefinitionsInfo(info);
                setVulnerabilityDefinitionsRequestHasError(false);
            })
            .catch(() => {
                setVulnerabilityDefinitionsInfo(null);
                setVulnerabilityDefinitionsRequestHasError(true);
            });
    }, [pollingCountSlower]);

    useInterval(() => {
        setPollingCountFaster(pollingCountFaster + 1);
    }, 30000); // 30 seconds is same as for Cluster Status Problems in top navigation

    useInterval(() => {
        setPollingCountSlower(pollingCountSlower + 1);
    }, 300000); // 5 minutes is enough for Vulnerability Definitions

    return (
        <section className="bg-primary-200 flex flex-col h-full overflow-auto relative">
            <PageHeader header="System Health" subHeader="Dashboard">
                <div className="flex flex-1 items-center justify-end">
                    <GenerateDiagnosticBundleButton />
                </div>
            </PageHeader>
            <div className="flex flex-col w-full px-4 py-2">
                <div className="grid grid-columns-1 md:grid-columns-3 grid-gap-4 py-2 w-full">
                    <Widget
                        className="sx-2"
                        header="Cluster Health"
                        headerComponents={<ViewAllButton url={clustersBasePath} />}
                        id="cluster-health"
                    >
                        {clustersRequestHasError ? (
                            <div className="p-2 w-full">
                                <Message type="error">Request failed for Clusters</Message>
                            </div>
                        ) : (
                            <div className="flex flex-wrap">
                                <Widget
                                    className="h-48 m-2 w-48"
                                    header="Cluster Overview"
                                    id="cluster-overview"
                                >
                                    <ClusterOverview clusters={clusters} />
                                </Widget>
                                <Widget
                                    className="h-48 m-2 text-center w-48"
                                    header="Collector Status"
                                    id="collector-status"
                                >
                                    <CollectorStatus clusters={clusters} />
                                </Widget>
                                <Widget
                                    className="h-48 m-2 text-center w-48"
                                    header="Admission Control Status"
                                    id="admissionControl-status"
                                >
                                    <AdmissionControlStatus clusters={clusters} />
                                </Widget>
                                <Widget
                                    className="h-48 m-2 text-center w-48"
                                    header="Sensor Status"
                                    id="sensor-status"
                                >
                                    <SensorStatus clusters={clusters} />
                                </Widget>
                                <Widget
                                    className="h-48 m-2 text-center w-48"
                                    header="Sensor Upgrade"
                                    id="sensor-upgrade"
                                >
                                    <SensorUpgrade clusters={clusters} />
                                </Widget>
                                <Widget
                                    className="h-48 m-2 text-center w-48"
                                    header="Credential Expiration"
                                    id="credential-expiration"
                                >
                                    <CredentialExpiration
                                        clusters={clusters}
                                        currentDatetime={currentDatetime}
                                    />
                                </Widget>
                            </div>
                        )}
                    </Widget>
                    <Widget
                        className="h-48 text-center"
                        header="Vulnerability Definitions"
                        id="vulnerability-definitions"
                    >
                        {vulnerabilityDefinitionsRequestHasError ? (
                            <div className="p-2 w-full">
                                <Message type="error">
                                    Request failed for Vulnerability Definitions
                                </Message>
                            </div>
                        ) : (
                            <VulnerabilityDefinitions
                                currentDatetime={currentDatetime}
                                vulnerabilityDefinitionsInfo={vulnerabilityDefinitionsInfo}
                            />
                        )}
                    </Widget>
                </div>
                <div className="grid grid-columns-1 md:grid-columns-3 grid-gap-4 py-2 w-full">
                    <ImageIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    <NotifierIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    <BackupIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    {isK8sAuditLoggingEnabled && (
                        <LogIntegrationHealthWidget pollingCount={pollingCountFaster} />
                    )}
                </div>
            </div>
        </section>
    );
};

export default SystemHealthDashboardPage;
