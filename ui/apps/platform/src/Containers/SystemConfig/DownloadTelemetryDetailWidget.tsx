/* eslint-disable jsx-a11y/control-has-associated-label */
import React, { ReactElement, useEffect, useState } from 'react';
import { Download, HelpCircle } from 'react-feather';
import { ClipLoader } from 'react-spinners';
import { parse } from 'date-fns';
import qs from 'qs';
import { Tooltip, DetailedTooltipOverlay } from '@stackrox/ui-components';

import Button from 'Components/Button';
import MultiSelect from 'Components/MultiSelect';

import { fetchClustersAsArray } from 'services/ClustersService';
import downloadDiagnostics from 'services/DebugService';

import FilterByStartingTimeValidationMessage from './FilterByStartingTimeValidationMessage';

// Recommended format:
const startingTimeFormat = 'yyyy-mm-ddThh:mmZ'; // seconds are optional but UTC is required

/* Minimal format:
 * requires year-month-day and hour-minute (does not exclude some invalid month-day combinations)
 * does not require seconds or thousandths
 * does require UTC as time zone
 */
export const startingTimeRegExp = /^20\d\d-(?:0\d|1[012])-(?:0[123456789]|1\d|2\d|3[01])T(?:0\d|1\d|2[0123]):[012345]\d(?::\d\d(?:\.\d\d\d)?)?Z$/;

type QueryStringProps = {
    selectedClusterNames: string[];
    startingTimeObject: Date | null;
    isStartingTimeValid: boolean;
};

export const getQueryString = ({
    selectedClusterNames,
    startingTimeObject,
    isStartingTimeValid,
}: QueryStringProps): string => {
    // The qs package ignores params which have undefined as value.
    const queryParams = {
        cluster: selectedClusterNames.length ? selectedClusterNames : undefined,
        since:
            startingTimeObject && isStartingTimeValid
                ? startingTimeObject.toISOString()
                : undefined,
    };

    return qs.stringify(queryParams, {
        addQueryPrefix: true, // except if empty string because all params are undefined
        arrayFormat: 'repeat', // for example, cluster=abbot&cluster=costello
        encodeValuesOnly: true,
    });
};

const inputBaseClassName =
    'bg-base-100 border-2 border-base-300 hover:border-base-400 font-600 leading-normal p-2 rounded text-base-600';

const titleHelp = 'Diagnostic Data Collection';
const subtitleHelp =
    'A bundle file contains data about system health of StackRox deployments in the Central cluster and currently connected secured clusters (either filter by clusters or include all clusters).';
const bodyItems = [
    'Heap profile of Central',
    'Database storage information (database size, free space on volume)',
    'Component health information for all StackRox components (version, memory usage, error conditions)',
    'Coarse-grained usage statistics (API endpoint invocation counts)',
    'Logs of all StackRox components limited in length by 50MB maximum size of the bundle file (either filter by starting time or default 20 minutes ago)',
    'Logs of recently crashed StackRox components from up to 20 minutes before the last crash',
    'Kubernetes YAML definitions of StackRox components (excluding Kubernetes secrets)',
    'Kubernetes events of objects in the StackRox namespaces',
    'Information about nodes in each secured cluster (kernel and OS versions, resource pressure, taints)',
    'Environment information about each secured cluster (Kubernetes version, if applicable cloud provider)',
];
/* eslint-disable react/no-array-index-key */
const bodyHelp = (
    <ul className="list-disc ml-4">
        {bodyItems.map((bodyItem, i) => (
            <li key={i}>{bodyItem}</li>
        ))}
    </ul>
);

type SelectOption = {
    label: string;
    value: string;
};

const DownloadTelemetryDetailWidget = (): ReactElement => {
    const [isDownloading, setIsDownloading] = useState<boolean>(false);

    const [availableClusterOptions, setAvailableClusterOptions] = useState<SelectOption[]>([]);
    const [selectedClusterNames, setSelectedClusterNames] = useState<string[]>([]);

    const [startingTimeText, setStartingTimeText] = useState<string>(''); // controlled input text
    const [startingTimeObject, setStartingTimeObject] = useState<Date | null>(null); // parsed from text
    const [isStartingTimeValid, setIsStartingTimeValid] = useState<boolean>(true);
    const [currentTimeObject, setCurrentTimeObject] = useState<Date | null>(null); // for pure message

    useEffect(() => {
        fetchClustersAsArray().then((clusters) => {
            setAvailableClusterOptions(clusters.map(({ name }) => ({ label: name, value: name })));
        });
    }, []);

    function onChangeStartingTime(event): void {
        const trimmedText = event.target.value.trim();
        setStartingTimeText(trimmedText);

        if (trimmedText.length === 0) {
            // This combination represents default starting time.
            setCurrentTimeObject(null);
            setStartingTimeObject(null);
            setIsStartingTimeValid(true);
        } else if (
            startingTimeRegExp.test(trimmedText) &&
            !Number.isNaN(Number(parse(trimmedText)))
        ) {
            const newTimeObject = new Date();
            const dateTimeObject = parse(trimmedText);

            setCurrentTimeObject(newTimeObject);
            setStartingTimeObject(dateTimeObject);

            // Successfully parsed text is valid if it is in the past.
            setIsStartingTimeValid(Number(dateTimeObject) < Number(newTimeObject));
        } else {
            // This combination represents unsuccessfully parsed text.
            setCurrentTimeObject(null);
            setStartingTimeObject(null);
            setIsStartingTimeValid(false);
        }
    }

    function triggerDownload(): void {
        setIsDownloading(true);
        const queryString = getQueryString({
            selectedClusterNames,
            startingTimeObject,
            isStartingTimeValid,
        });
        downloadDiagnostics(queryString).finally(() => {
            setIsDownloading(false);
        });
    }

    const icon = (
        <div className="mr-2">
            {isDownloading ? (
                <ClipLoader color="currentColor" loading size={15} />
            ) : (
                <Download className="h-4 w-4" />
            )}
        </div>
    );

    // TODO Investigate why data-testid attribute does not work for MultiSelect.
    return (
        <div className="px-3 w-full h-full" data-testid="download-telemetry">
            <div className="bg-base-100 border-base-200 shadow h-full">
                <h2 className="py-2 px-4 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between items-center h-10">
                    Diagnostic Data
                </h2>

                <div className="flex flex-col pt-4 pb-4 px-4 w-full">
                    <div className="w-full pr-4 whitespace-pre-wrap leading-normal">
                        <div className="pb-2 text-base-600 leading-normal">
                            Diagnostic data is available for download in the ZIP format. If you wish
                            to inspect the data before sending it to our Customer Success team,
                            please use an appropriate program to view its contents.
                        </div>
                        <div className="pb-4 text-base-600 leading-normal">
                            You can filter which data to include in the bundle (maximum size 50MB).
                        </div>
                        <div className="pb-4" data-testid="filter-by-clusters">
                            <div className="pb-2">
                                <span className="text-base-700">Filter by clusters</span>
                            </div>
                            <MultiSelect
                                className=""
                                name="filterByClusters"
                                onChange={setSelectedClusterNames}
                                options={availableClusterOptions}
                                placeholder="No clusters selected means include all clusters"
                                value={selectedClusterNames}
                            />
                        </div>
                        <div className="pb-4">
                            <div className="pb-2">
                                <label htmlFor="filterByStartingTime" className="text-base-700">
                                    Filter by starting time (seconds are optional but UTC is
                                    required)
                                </label>
                            </div>
                            <div className="flex flex-row items-center">
                                <input
                                    type="text"
                                    id="filterByStartingTime"
                                    name="filterByStartingTime"
                                    onChange={onChangeStartingTime}
                                    placeholder={startingTimeFormat}
                                    className={`${inputBaseClassName} mr-4 w-48`}
                                    value={startingTimeText}
                                />
                                <FilterByStartingTimeValidationMessage
                                    currentTimeObject={currentTimeObject}
                                    isStartingTimeValid={isStartingTimeValid}
                                    startingTimeFormat={startingTimeFormat}
                                    startingTimeObject={startingTimeObject}
                                />
                            </div>
                        </div>
                        <div className="flex flex-row items-center">
                            <Button
                                icon={icon}
                                className="btn btn-tertiary"
                                onClick={triggerDownload}
                                disabled={isDownloading || !isStartingTimeValid}
                                dataTestId="download-diagnostic-data"
                                text="Download diagnostic data"
                            />
                            <Tooltip
                                content={
                                    <DetailedTooltipOverlay
                                        title={titleHelp}
                                        subtitle={subtitleHelp}
                                        body={bodyHelp}
                                    />
                                }
                            >
                                <HelpCircle className="h-4 w-4 ml-4 text-primary-700" />
                            </Tooltip>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

DownloadTelemetryDetailWidget.propTypes = {};

export default DownloadTelemetryDetailWidget;
