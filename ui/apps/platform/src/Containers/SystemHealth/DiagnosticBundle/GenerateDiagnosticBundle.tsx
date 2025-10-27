import { useState } from 'react';
import type { ReactElement } from 'react';
import { Button, Flex, Modal, ModalBoxBody, ModalBoxFooter } from '@patternfly/react-core';
import { DownloadIcon } from '@patternfly/react-icons';
import { useFormik } from 'formik';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useModal from 'hooks/useModal';
import useMetadata from 'hooks/useMetadata';
import downloadDiagnostics from 'services/DebugService';
import type { DiagnosticBundleRequest } from 'services/DebugService';
import { getVersionedDocs } from 'utils/versioning';

import DiagnosticBundleForm from './DiagnosticBundleForm';
import { getQueryString } from './diagnosticBundleUtils';

const initialValues: DiagnosticBundleRequest = {
    filterByClusters: [],
    isDatabaseDiagnosticsOnly: false,
    includeComplianceOperatorResources: false,
    startingDate: '',
    startingTime: '',
};

function GenerateDiagnosticBundle(): ReactElement {
    const { isModalOpen, openModal, closeModal } = useModal();
    const { version } = useMetadata();

    const { submitForm, setFieldValue, values, handleBlur, isSubmitting, setSubmitting } =
        useFormik({
            initialValues,
            onSubmit: triggerDownload,
        });

    const startingTimeIso: string | null = React.useMemo(() => {
        if (!values.startingDate) {
            return null;
        }
        const time = values.startingTime || '00:00';
        return `${values.startingDate}T${time}:00.000Z`;
    }, [values.startingDate, values.startingTime]);

    function triggerDownload(): void {
        const queryString = getQueryString({
            selectedClusterNames: values.filterByClusters,
            startingTimeIso,
            isDatabaseDiagnosticsOnly: values.isDatabaseDiagnosticsOnly,
            includeComplianceOperatorResources: values.includeComplianceOperatorResources,
        });
        downloadDiagnostics(queryString)
            .catch(() => {
                // TODO render error in DiagnosticBundleForm
            })
            .finally(() => {
                setSubmitting(false);
            });
    }

    return (
        <>
            <Button
                key="open-diagnostic-bundle-modal"
                data-testid="diagnostic-bundle-modal-open-button"
                variant="secondary"
                onClick={openModal}
            >
                Generate diagnostic bundle
            </Button>
            <Modal
                title="Diagnostic bundle"
                description="You can filter which platform data to include in the Zip file (max size 50MB)"
                isOpen={isModalOpen}
                variant="medium"
                onClose={closeModal}
                aria-label="Diagnostic bundle"
                hasNoBodyWrapper
            >
                <ModalBoxBody>
                    <DiagnosticBundleForm
                        values={values}
                        setFieldValue={setFieldValue}
                        handleBlur={handleBlur}
                    />
                </ModalBoxBody>
                <ModalBoxFooter>
                    <Flex spaceItems={{ default: 'spaceItemsLg' }}>
                        <Button
                            variant="primary"
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
                                        'configuring/generate-diagnostic-bundle'
                                    )}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    Generate a diagnostic bundle
                                </a>
                            </ExternalLink>
                        )}
                    </Flex>
                </ModalBoxFooter>
            </Modal>
        </>
    );
}

export default GenerateDiagnosticBundle;
