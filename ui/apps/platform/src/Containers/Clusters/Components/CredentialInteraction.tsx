import React, { ReactElement, useState } from 'react';

import { generateSecuredClusterCertSecret } from 'services/CertGenerationService';
import { rotateClusterCerts } from 'services/ClustersService';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';

import CredentialExpiration from './CredentialExpiration';
import {
    findUpgradeState,
    initiationOfCertRotationIfApplicable,
    isCertificateExpiringSoon,
    isUpToDateStateObject,
} from '../cluster.helpers';
import { CertExpiryStatus, SensorUpgradeStatus } from '../clusterTypes';

/*
 * The heading is a simple explanation of what did not happen because of the error.
 */
function getErrorElement(heading: string, error): ReactElement {
    return (
        <div className="mt-2">
            <div className="font-700" data-testid="reissue-error-heading">
                {heading}
            </div>
            {error?.response?.data?.message && (
                <div data-testid="reissue-error-message">{error.response.data.message}</div>
            )}
        </div>
    );
}

const download = 'download';
const upgrade = 'upgrade';

const fieldClassName = 'flex flex-row items-center mb-2';
const radioClassName = 'flex-shrink-0 h-4 w-4';
const labelClassName = 'leading-tight ml-2';

/*
 * Credential Interaction in Cluster side panel.
 *
 * Display either the form to reissue certificate or an explanation of the result.
 */

type CredentialInteractionProps = {
    certExpiryStatus: CertExpiryStatus;
    upgradeStatus: SensorUpgradeStatus;
    clusterId: string;
};

function CredentialInteraction({
    certExpiryStatus,
    upgradeStatus,
    clusterId,
}: CredentialInteractionProps): ReactElement {
    const upgradeStateObject = findUpgradeState(upgradeStatus);
    const isUpToDate = isUpToDateStateObject(upgradeStateObject);
    const currentDatetime = new Date();

    const [howToReissue, setHowToReissue] = useState(isUpToDate ? upgrade : download);
    const [disabledReissueButton, setDisabledReissueButton] = useState(false);
    const [errorElement, setErrorElement] = useState(<></>);

    const [isDownloadSuccessful, setIsDownloadSuccessful] = useState(false);

    let interactionElement = <></>;

    if (isDownloadSuccessful) {
        interactionElement = (
            <div className="mt-2">
                <div data-testid="downloadedToReissueCertificate">
                    Apply downloaded YAML file to the cluster:{' '}
                    <span className="font-700 whitespace-nowrap">kubectl apply -f</span>
                </div>
                <div>
                    Sensor, Admission Controller, and Collectors begin using new credentials the
                    next time they restart.
                </div>
            </div>
        );
    } else {
        const datetimeOfCertRotation = initiationOfCertRotationIfApplicable(upgradeStatus);

        if (datetimeOfCertRotation) {
            // Order arguments according to date-fns@2 convention:
            // If initiationOfCertRotation <= currentDateTime: X units ago
            interactionElement = (
                <div className="mt-2">
                    <div data-testid="upgradedToReissueCertificate">
                        An automatic upgrade applied new credentials to the cluster{' '}
                        {getDistanceStrictAsPhrase(datetimeOfCertRotation, currentDatetime)}.
                    </div>
                    <div>
                        Sensor, Admission Controller, and Collectors begin using new credentials the
                        next time they restart.
                    </div>
                </div>
            );
        } else if (isCertificateExpiringSoon(certExpiryStatus, currentDatetime)) {
            const onChangeHowToReissue = (event) => {
                setHowToReissue(event.target.value);
            };

            const onClickReissue = () => {
                if (howToReissue === download) {
                    setDisabledReissueButton(true);
                    generateSecuredClusterCertSecret(clusterId)
                        .then(() => {
                            setIsDownloadSuccessful(true);
                        })
                        .catch((error) => {
                            setErrorElement(
                                getErrorElement('Failed to regenerate certificates', error)
                            );
                        })
                        .finally(() => {
                            setDisabledReissueButton(false);
                        });
                } else if (howToReissue === upgrade) {
                    setDisabledReissueButton(true);
                    rotateClusterCerts(clusterId)
                        .catch((error) => {
                            setErrorElement(
                                getErrorElement(
                                    'Failed to apply new credentials to the cluster',
                                    error
                                )
                            );
                        })
                        .finally(() => {
                            setDisabledReissueButton(false);
                        });
                }
            };

            interactionElement = (
                <form className="mt-2">
                    <ul>
                        <li className={fieldClassName}>
                            <input
                                type="radio"
                                id="downloadToReissueCertificate"
                                data-testid="downloadToReissueCertificate"
                                name="howToReissue"
                                value={download}
                                checked={howToReissue === download}
                                onChange={onChangeHowToReissue}
                                className={radioClassName}
                            />
                            <label
                                htmlFor="downloadToReissueCertificate"
                                className={labelClassName}
                            >
                                Download YAML file
                                <br />
                                and then apply it to the cluster
                            </label>
                        </li>
                        <li className={fieldClassName}>
                            <input
                                type="radio"
                                id="upgradeToReissueCertificate"
                                data-testid="upgradeToReissueCertificate"
                                name="howToReissue"
                                value={upgrade}
                                checked={howToReissue === upgrade}
                                onChange={onChangeHowToReissue}
                                className={radioClassName}
                                disabled={!isUpToDate}
                            />
                            <label htmlFor="upgradeToReissueCertificate" className={labelClassName}>
                                Use automatic upgrade
                                <br />
                                if Sensor is up to date with Central
                            </label>
                        </li>
                    </ul>
                    <button
                        type="button"
                        disabled={disabledReissueButton}
                        onClick={onClickReissue}
                        className="btn btn-tertiary"
                        data-testid="reissueCertificateButton"
                    >
                        Re-issue certificate
                    </button>
                </form>
            );
        }
    }

    return (
        <div className="flex flex-col">
            <CredentialExpiration certExpiryStatus={certExpiryStatus} />
            {interactionElement}
            {errorElement}
        </div>
    );
}

export default CredentialInteraction;
