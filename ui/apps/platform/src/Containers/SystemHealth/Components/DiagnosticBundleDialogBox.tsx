import React, { ReactElement, useEffect, useState } from 'react';
import { Download, ExternalLink } from 'react-feather';
import { ClipLoader } from 'react-spinners';
import { parse } from 'date-fns';
import qs from 'qs';

import Button from 'Components/Button';
import MultiSelect from 'Components/MultiSelect';

import { fetchClustersAsArray } from 'services/ClustersService';
import downloadDiagnostics from 'services/DebugService';

import { Alert, AlertVariant } from '@patternfly/react-core';
import useMetadata from 'hooks/useMetadata';
import { getVersionedDocs } from 'utils/versioning';
import FilterByStartingTimeValidationMessage from './FilterByStartingTimeValidationMessage';

// Recommended format:
const startingTimeFormat = 'yyyy-mm-ddThh:mmZ'; // seconds are optional but UTC is required

/* Minimal format:
 * requires year-month-day and hour-minute (does not exclude some invalid month-day combinations)
 * does not require seconds or thousandths
 * does require UTC as time zone
 */
export const startingTimeRegExp =
    /^20\d\d-(?:0\d|1[012])-(?:0[123456789]|1\d|2\d|3[01])T(?:0\d|1\d|2[0123]):[012345]\d(?::\d\d(?:\.\d\d\d)?)?Z$/;

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

type SelectOption = {
    label: string;
    value: string;
};

const DiagnosticBundleDialogBox = (): ReactElement => {
    const [isDownloading, setIsDownloading] = useState<boolean>(false);

    const [availableClusterOptions, setAvailableClusterOptions] = useState<SelectOption[]>([]);
    const [selectedClusterNames, setSelectedClusterNames] = useState<string[]>([]);

    const [startingTimeText, setStartingTimeText] = useState<string>(''); // controlled input text
    const [startingTimeObject, setStartingTimeObject] = useState<Date | null>(null); // parsed from text
    const [isStartingTimeValid, setIsStartingTimeValid] = useState<boolean>(true);
    const [currentTimeObject, setCurrentTimeObject] = useState<Date | null>(null); // for pure message

    const [alertDownload, setAlertDownload] = useState<ReactElement | null>(null);

    const { version } = useMetadata();

    useEffect(() => {
        fetchClustersAsArray()
            .then((clusters) => {
                setAvailableClusterOptions(
                    clusters.map(({ name }) => ({ label: name, value: name }))
                );
            })
            .catch(() => {
                // TODO display message when there is a place for minor errors
            });
    }, []);

    function onChangeStartingTime(event: React.ChangeEvent<HTMLInputElement>): void {
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
        downloadDiagnostics(queryString)
            .then(() => {
                setAlertDownload(null);
            })
            .catch((error) => {
                setAlertDownload(
                    <Alert
                        title="Downloading diagnostic bundle failed."
                        variant={AlertVariant.danger}
                        isInline
                    >
                        <p>
                            <b>{error.message}</b>
                            <br />
                            If timeout is exceeded, use `roxctl` command line tool with increased
                            timeout instead: `roxctl central debug download-diagnostics
                            --timeout=400s`
                        </p>
                    </Alert>
                );
            })
            .finally(() => {
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
    // The width is enough for content and avoids too much overlap with System Health.
    return (
        <div
            className="bg-base-100 border-2 border-base-400 shadow"
            style={{ width: '35rem' }}
            data-testid="diagnostic-bundle-dialog-box"
        >
            <div className="border-b border-base-400 flex font-700 items-center h-10 leading-normal px-2 text-base-600 text-sm tracking-wide uppercase">
                Diagnostic Bundle
            </div>
            {alertDownload}
            <form className="border-base-300 flex flex-col leading-normal p-2 text-base-600 w-full">
                <div className="pb-4">
                    You can filter which platform data to include in the Zip file (max size 50MB)
                </div>
                <div className="pb-4" data-testid="filter-by-clusters">
                    <div className="pb-2">
                        <span className="font-700">Filter by clusters</span>
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
                        <label htmlFor="filter-by-starting-time">
                            <span className="font-700">Filter by starting time</span>{' '}
                            <span>(seconds are optional but UTC is required)</span>
                        </label>
                    </div>
                    <div className="flex flex-row items-center">
                        <input
                            type="text"
                            id="filter-by-starting-time"
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
                <div className="flex flex-row items-center justify-between">
                    <Button
                        icon={icon}
                        className="btn btn-tertiary"
                        onClick={triggerDownload}
                        disabled={isDownloading || !isStartingTimeValid}
                        dataTestId="download-diagnostic-bundle-button"
                        text="Download Diagnostic Bundle"
                    />
                    {version && (
                        <div className="inline-flex flex-row text-tertiary-700">
                            <a
                                href={getVersionedDocs(
                                    version,
                                    'configuration/generate-diagnostic-bundle.html'
                                )}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="underline"
                            >
                                Generate a diagnostic bundle
                            </a>
                            <span className="flex-shrink-0 ml-2">
                                <ExternalLink className="h-4 w-4" />
                            </span>
                        </div>
                    )}
                </div>
            </form>
        </div>
    );
};

export default DiagnosticBundleDialogBox;
