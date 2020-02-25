import React, { useState } from 'react';
import downloadDiagnostics from 'services/DebugService';
import PanelButton from 'Components/PanelButton';
import * as Icon from 'react-feather';
import { ClipLoader } from 'react-spinners';

const DownloadTelemetryDetailWidget = () => {
    const [isDownloading, setIsDownloading] = useState(false);

    function triggerDownload() {
        setIsDownloading(true);
        downloadDiagnostics().finally(() => {
            setIsDownloading(false);
        });
    }

    const icon = isDownloading ? (
        <ClipLoader color="blue" loading size={15} />
    ) : (
        <Icon.Download className="h-4 w-4 ml-1 text-primary-600" />
    );

    return (
        <div className="px-3 w-full h-full" data-test-id="download-telemetry">
            <div className="bg-base-100 border-base-200 shadow h-full">
                <h2 className="py-2 px-4 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between items-center h-10">
                    Diagnostic Data{' '}
                </h2>

                <div className="flex flex-col pt-2 pb-4 px-4 w-full">
                    <div className="w-full pr-4 whitespace-pre-wrap leading-normal">
                        <div className="py-2 text-base-600 font-700">
                            Diagnostic data is available for download in the ZIP format. If you wish
                            to inspect the data before sending it to our Customer Success team,
                            please use an appropriate program to view its contents.
                        </div>
                        <div className="flex justify-center">
                            <PanelButton
                                icon={icon}
                                className="flex leading-normal rounded font-700 text-sm text-base-600 no-underline py-4 px-1 items-center uppercase border-2 border-base-300 shadow"
                                onClick={triggerDownload}
                                disabled={isDownloading}
                                alwaysVisibleText
                                tooltip={
                                    <div className="w-auto">
                                        <h3 className="mb-2 font-700 text-lg uppercase">
                                            What we collect:
                                        </h3>
                                        <p className="mb-2">
                                            The diagnostic bundle contains information pertaining to
                                            the system health of the StackRox deployments
                                            <br /> in the central cluster as well as all currently
                                            connected secured clusters.
                                        </p>
                                        <p className="mb-1">It includes:</p>
                                        <ul className="mb-1 w-full list-disc ml-4">
                                            <li>Heap profile of Central</li>
                                            <li>
                                                Database storage information (database size, free
                                                space on volume)
                                            </li>
                                            <li>
                                                Component health information for all StackRox
                                                components (version, memory usage, error conditions)
                                            </li>
                                            <li>
                                                Coarse-grained usage statistics (API endpoint
                                                invocation counts)
                                            </li>
                                            <li>
                                                Logs of all StackRox components from the last 20
                                                minutes
                                            </li>
                                            <li>
                                                Logs of recently crashed StackRox components from up
                                                to 20 minutes before the last crash
                                            </li>
                                            <li>
                                                Kubernetes YAML definitions of StackRox components
                                                (excluding Kubernetes secrets)
                                            </li>
                                            <li>
                                                Kubernetes events of objects in the StackRox
                                                namespaces
                                            </li>
                                            <li>
                                                Information about nodes in each secured cluster
                                                (kernel and OS versions, resource pressure, taints)
                                            </li>
                                            <li>
                                                Environment information about each secured cluster
                                                (Kubernetes version, if applicable cloud provider)
                                            </li>
                                        </ul>
                                    </div>
                                }
                            >
                                Download diagnostic data{' '}
                                <Icon.HelpCircle className="h-4 w-4 text-primary-400 ml-2" />
                            </PanelButton>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

DownloadTelemetryDetailWidget.propTypes = {};

export default DownloadTelemetryDetailWidget;
