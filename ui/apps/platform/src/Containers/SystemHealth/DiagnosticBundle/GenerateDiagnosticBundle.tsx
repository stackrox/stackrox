import React, { useState, ReactElement } from 'react';
import { Button, ButtonVariant, Flex, Popover, PopoverPosition } from '@patternfly/react-core';
import { AngleDownIcon, AngleUpIcon, DownloadIcon } from '@patternfly/react-icons';
import { useFormik } from 'formik';
import { parse } from 'date-fns';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useMetadata from 'hooks/useMetadata';
import downloadDiagnostics, { DiagnosticBundleRequest } from 'services/DebugService';
import { getVersionedDocs } from 'utils/versioning';

import DiagnosticBundleForm from './DiagnosticBundleForm';
import { getQueryString, startingTimeRegExp } from './diagnosticBundleUtils';

const initialValues: DiagnosticBundleRequest = {
    filterByClusters: [],
    filterByStartingTime: '',
};

function GenerateDiagnosticBundle(): ReactElement {
    const [isOpen, setIsOpen] = useState<boolean>(false);
    const [startingTimeObject, setStartingTimeObject] = useState<Date | null>(null); // parsed from text
    const [isStartingTimeValid, setIsStartingTimeValid] = useState<boolean>(true);
    const [currentTimeObject, setCurrentTimeObject] = useState<Date | null>(null); // for pure message
    const { version } = useMetadata();

    function onChangeStartingTime(event: React.FormEvent<HTMLInputElement>): void {
        const trimmedText = event.currentTarget.value.trim();

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

    const { submitForm, setFieldValue, values, handleBlur, isSubmitting, setSubmitting } =
        useFormik({
            initialValues,
            onSubmit: triggerDownload,
        });

    function triggerDownload(): void {
        const queryString = getQueryString({
            selectedClusterNames: values.filterByClusters,
            startingTimeObject,
            isStartingTimeValid,
        });
        downloadDiagnostics(queryString)
            .catch(() => {
                // TODO render error in DiagnosticBundleForm
            })
            .finally(() => {
                setSubmitting(false);
            });
    }

    const footerContent = (
        <Flex spaceItems={{ default: 'spaceItemsLg' }}>
            <Button
                variant={ButtonVariant.primary}
                onClick={submitForm}
                icon={isSubmitting ? null : <DownloadIcon />}
                spinnerAriaValueText={isSubmitting ? 'Downloading' : undefined}
                isLoading={isSubmitting}
            >
                Download diagnostic bundle
            </Button>
            {version && (
                <ExternalLink>
                    <a
                        href={getVersionedDocs(
                            version,
                            'configuration/generate-diagnostic-bundle.html'
                        )}
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        Generate a diagnostic bundle
                    </a>
                </ExternalLink>
            )}
        </Flex>
    );

    return (
        <Popover
            aria-label="Choose options to generate a diagnostic bundle"
            headerContent={<h2>Diagnostic bundle</h2>}
            bodyContent={
                <DiagnosticBundleForm
                    values={values}
                    setFieldValue={setFieldValue}
                    handleBlur={handleBlur}
                    currentTimeObject={currentTimeObject}
                    startingTimeObject={startingTimeObject}
                    isStartingTimeValid={isStartingTimeValid}
                    onChangeStartingTime={onChangeStartingTime}
                />
            }
            footerContent={footerContent}
            maxWidth="100%"
            position={PopoverPosition.bottomEnd}
            shouldOpen={() => setIsOpen(true)}
            shouldClose={() => setIsOpen(false)}
            showClose={false}
            isVisible={isOpen}
        >
            <Button variant={ButtonVariant.secondary}>
                <Flex
                    alignItems={{ default: 'alignItemsCenter' }}
                    spaceItems={{ default: 'spaceItemsXs' }}
                >
                    <span>Generate diagnostic bundle</span>
                    {isOpen ? <AngleUpIcon /> : <AngleDownIcon />}
                </Flex>
            </Button>
        </Popover>
    );
}

export default GenerateDiagnosticBundle;
