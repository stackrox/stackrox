import React, { ReactElement, useEffect, useState } from 'react';
import {
    Alert,
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    FlexItem,
} from '@patternfly/react-core';
import { DownloadIcon, ExternalLinkAltIcon } from '@patternfly/react-icons';

import useMetadata from 'hooks/useMetadata';
import usePermissions from 'hooks/usePermissions';
import { generateCertSecretForComponent } from 'services/CertGenerationService';
import { fetchCertExpiryForComponent } from 'services/CredentialExpiryService';
import { CertExpiryComponent } from 'types/credentialExpiryService.proto';
import {
    getCredentialExpiryPhrase,
    getCredentialExpiryVariant,
    nameOfComponent,
} from 'utils/credentialExpiry';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getVersionedDocs } from 'utils/versioning';

import { ErrorIcon, healthIconMap, SpinnerIcon } from '../CardHeaderIcons';

type CertificateCardProps = {
    component: CertExpiryComponent;
    pollingCount: number;
};

function CertificateCard({ component, pollingCount }: CertificateCardProps): ReactElement {
    const [isFetching, setIsFetching] = useState(false);
    const [errorMessageFetching, setErrorMessageFetching] = useState('');
    const [expirationDate, setExpirationDate] = useState('');

    const [currentDatetime, setCurrentDatetime] = useState<Date | null>(null);

    const [isDownloading, setIsDownloading] = useState(false);
    const [errorMessageDownloading, setErrorMessageDownloading] = useState('');

    const { hasReadWriteAccess } = usePermissions();
    const hasAdministrationWritePermission = hasReadWriteAccess('Administration');

    const { version } = useMetadata();

    useEffect(() => {
        setIsFetching(true);
        fetchCertExpiryForComponent(component)
            .then((expiry) => {
                setErrorMessageFetching('');
                setExpirationDate(expiry);
                setCurrentDatetime(new Date());
            })
            .catch((error) => {
                setErrorMessageFetching(getAxiosErrorMessage(error));
                setExpirationDate('');
                setCurrentDatetime(null);
            })
            .finally(() => {
                setIsFetching(false);
            });
    }, [component, pollingCount]);

    function onDownload() {
        setIsDownloading(true);
        generateCertSecretForComponent(component)
            .then(() => {
                setErrorMessageDownloading('');
            })
            .catch((error) => {
                setErrorMessageDownloading(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsDownloading(false);
            });
    }

    const title = `${nameOfComponent[component]} certificate`;

    /*
     * Wait for isFetching only until response to the initial request.
     * Otherwise phrase temporarily disappears during each subsequent request.
     */
    const isFetchingInitialRequest = isFetching && pollingCount === 0;

    /* eslint-disable no-nested-ternary */
    const icon = isFetchingInitialRequest
        ? SpinnerIcon
        : !expirationDate || !currentDatetime
          ? ErrorIcon
          : healthIconMap[getCredentialExpiryVariant(expirationDate, currentDatetime)];

    return (
        <Card isCompact>
            <CardHeader>
                <Flex className="pf-u-flex-grow-1">
                    <FlexItem>{icon}</FlexItem>
                    <FlexItem>
                        <CardTitle component="h2">{title}</CardTitle>
                    </FlexItem>
                    {currentDatetime && expirationDate && (
                        <FlexItem>
                            {getCredentialExpiryPhrase(expirationDate, currentDatetime)}
                        </FlexItem>
                    )}
                </Flex>
            </CardHeader>
            <CardBody>
                {hasAdministrationWritePermission ? (
                    <Flex>
                        <FlexItem>
                            To update the certificate, download the YAML file and apply it to your
                            cluster
                        </FlexItem>
                        <FlexItem>
                            <Button
                                variant="secondary"
                                icon={<DownloadIcon />}
                                isDisabled={isDownloading}
                                isLoading={isDownloading}
                                isSmall
                                onClick={onDownload}
                            >
                                Download YAML
                            </Button>
                        </FlexItem>
                        {version && (
                            <FlexItem align={{ default: 'alignRight' }}>
                                <Button
                                    variant="link"
                                    isInline
                                    component="a"
                                    href={getVersionedDocs(
                                        version,
                                        'configuration/reissue-internal-certificates.html'
                                    )}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    <Flex
                                        alignItems={{ default: 'alignItemsCenter' }}
                                        spaceItems={{ default: 'spaceItemsSm' }}
                                    >
                                        <span>Reissuing internal certificates</span>
                                        <ExternalLinkAltIcon color="var(--pf-global--link--Color)" />
                                    </Flex>
                                </Button>
                            </FlexItem>
                        )}
                    </Flex>
                ) : expirationDate &&
                  currentDatetime &&
                  getCredentialExpiryVariant(expirationDate, currentDatetime) !== 'success' ? (
                    <Flex>
                        <FlexItem>
                            To update the certificate, please contact your administrator
                        </FlexItem>
                    </Flex>
                ) : null}
                {errorMessageFetching && (
                    <Alert
                        isInline
                        variant="warning"
                        title={errorMessageFetching}
                        className="pf-u-mt-md"
                    />
                )}
                {errorMessageDownloading && (
                    <Alert
                        isInline
                        variant="danger"
                        title={errorMessageDownloading}
                        className="pf-u-mt-md"
                    />
                )}
            </CardBody>
        </Card>
    );
    /* eslint-enable no-nested-ternary */
}

export default CertificateCard;
