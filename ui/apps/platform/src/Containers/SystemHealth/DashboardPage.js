import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { HashLink } from 'react-router-hash-link';

import Message from 'Components/Message';
import PageHeader from 'Components/PageHeader';
import Widget from 'Components/Widget';
import integrationsList from 'Containers/Integrations/integrationsList';
import useInterval from 'hooks/useInterval';
import { clustersPath, integrationsPath } from 'routePaths';
import { fetchClustersAsArray } from 'services/ClustersService';
import {
    fetchBackupIntegrationsHealth,
    fetchImageIntegrationsHealth,
    fetchPluginIntegrationsHealth,
    fetchVulnerabilityDefinitionsInfo,
} from 'services/IntegrationHealthService';
import { fetchIntegration } from 'services/IntegrationsService';

import ClusterOverview from './Components/ClusterOverview';
import CollectorStatus from './Components/CollectorStatus';
import CredentialExpiration from './Components/CredentialExpiration';
import GenerateDiagnosticBundleButton from './Components/GenerateDiagnosticBundleButton';
import IntegrationsHealth from './Components/IntegrationsHealth';
import SensorStatus from './Components/SensorStatus';
import SensorUpgrade from './Components/SensorUpgrade';
import VulnerabilityDefinitions from './Components/VulnerabilityDefinitions';

import { mergeIntegrationResponses } from './utils/integrations';

const smallButtonClassName = 'btn-sm btn-base flex-shrink-0 no-underline whitespace-no-wrap';

const SystemHealthDashboardPage = () => {
    const [pollingCountFaster, setPollingCountFaster] = useState(0);
    const [pollingCountSlower, setPollingCountSlower] = useState(0);
    const [currentDatetime, setCurrentDatetime] = useState(null);
    const [clusters, setClusters] = useState([]);
    const [backupIntegrationsMerged, setBackupIntegrationsMerged] = useState([]);
    const [imageIntegrationsMerged, setImageIntegrationsMerged] = useState([]);
    const [pluginIntegrationsMerged, setPluginIntegrationsMerged] = useState([]);
    const [vulnerabilityDefinitionsInfo, setVulnerabilityDefinitionsInfo] = useState(null);
    const [vulnerabilityDefinitionsHasError, setVulnerabilityDefinitionsHasError] = useState(false);

    useEffect(() => {
        setCurrentDatetime(new Date());
        fetchClustersAsArray().then((array) => {
            setClusters(array);
        });
        // TODO catch
        Promise.all([fetchBackupIntegrationsHealth(), fetchIntegration('backups')]).then(
            ([integrationsHealth, { response }]) => {
                setBackupIntegrationsMerged(
                    mergeIntegrationResponses(
                        integrationsHealth,
                        response.externalBackups,
                        integrationsList.backups
                    )
                );
            }
        );
        Promise.all([fetchImageIntegrationsHealth(), fetchIntegration('imageIntegrations')]).then(
            ([integrationsHealth, { response }]) => {
                setImageIntegrationsMerged(
                    mergeIntegrationResponses(
                        integrationsHealth,
                        response.integrations,
                        integrationsList.imageIntegrations
                    )
                );
            }
        );
        Promise.all([fetchPluginIntegrationsHealth(), fetchIntegration('notifiers')]).then(
            ([integrationsHealth, { response }]) => {
                setPluginIntegrationsMerged(
                    mergeIntegrationResponses(
                        integrationsHealth,
                        response.notifiers,
                        integrationsList.plugins
                    )
                );
            }
        );
    }, [pollingCountFaster]);

    useEffect(() => {
        fetchVulnerabilityDefinitionsInfo()
            .then((info) => {
                setVulnerabilityDefinitionsInfo(info);
                setVulnerabilityDefinitionsHasError(false);
            })
            .catch(() => {
                setVulnerabilityDefinitionsInfo(null);
                setVulnerabilityDefinitionsHasError(true);
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
                        headerComponents={
                            <Link to={clustersPath} className={smallButtonClassName}>
                                View All
                            </Link>
                        }
                        id="cluster-health"
                    >
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
                    </Widget>
                    <Widget
                        className="h-48 text-center"
                        header="Vulnerability Definitions"
                        id="vulnerability-definitions"
                    >
                        {vulnerabilityDefinitionsHasError ? (
                            <div className="p-2 w-full">
                                <Message
                                    type="error"
                                    message="An error occurred requesting Vulnerability Definitions"
                                />
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
                    <Widget
                        header="Image Integrations"
                        headerComponents={
                            <HashLink
                                to={`${integrationsPath}#image-integrations`}
                                className={smallButtonClassName}
                            >
                                View All
                            </HashLink>
                        }
                        id="image-integrations"
                    >
                        <IntegrationsHealth integrationsMerged={imageIntegrationsMerged} />
                    </Widget>
                    <Widget
                        header="Plugin Integrations"
                        headerComponents={
                            <HashLink
                                to={`${integrationsPath}#plugin-integrations`}
                                className={smallButtonClassName}
                            >
                                View All
                            </HashLink>
                        }
                        id="plugin-integrations"
                    >
                        <IntegrationsHealth integrationsMerged={pluginIntegrationsMerged} />
                    </Widget>
                    <Widget
                        header="Backup Integrations"
                        headerComponents={
                            <HashLink
                                to={`${integrationsPath}#backup-integrations`}
                                className={smallButtonClassName}
                            >
                                View All
                            </HashLink>
                        }
                        id="backup-integrations"
                    >
                        <IntegrationsHealth integrationsMerged={backupIntegrationsMerged} />
                    </Widget>
                </div>
            </div>
        </section>
    );
};

export default SystemHealthDashboardPage;
