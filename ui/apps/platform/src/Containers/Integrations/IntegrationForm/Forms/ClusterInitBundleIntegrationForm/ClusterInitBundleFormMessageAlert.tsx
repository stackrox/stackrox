import React, { ReactElement } from 'react';
import { Alert, TextContent, Button, Flex, FlexItem } from '@patternfly/react-core';
import FileSaver from 'file-saver';

import { ClusterInitBundle } from 'services/ClustersService';

import ClusterInitBundleDetails from './ClusterInitBundleDetails';

export type ClusterInitBundleResponse = {
    meta: ClusterInitBundle;
    helmValuesBundle: string;
    kubectlBundle: string;
};

const onDownloadHandler = (fileName: string, currentBundle: string) => () => {
    const decodedValuesBundle = typeof currentBundle === 'string' ? atob(currentBundle) : '';

    const file = new Blob([decodedValuesBundle], {
        type: 'application/x-yaml',
    });

    FileSaver.saveAs(file, fileName);
};

function ClusterInitBundleResponseDetails({
    meta,
    helmValuesBundle,
    kubectlBundle,
}: ClusterInitBundleResponse) {
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXl' }}>
            <FlexItem>
                <TextContent className="pf-u-mb-md">
                    Please copy the generated cluster init bundle YAML file and store it safely. You
                    will not be able to access it again after you close this window.
                </TextContent>
                <Button
                    variant="secondary"
                    onClick={onDownloadHandler(
                        `${meta.name}-cluster-init-bundle.yaml`,
                        helmValuesBundle
                    )}
                >
                    Download Helm values file
                </Button>
            </FlexItem>
            <FlexItem>
                <TextContent className="pf-u-mb-md">
                    Use the following file if you do not want your secrets to be managed by Helm.
                    Most users should use the Helm values file above instead.
                </TextContent>
                <Button
                    variant="secondary"
                    onClick={onDownloadHandler(
                        `${meta.name}-cluster-init-secrets.yaml`,
                        kubectlBundle
                    )}
                >
                    Download Kubernetes secrets file
                </Button>
            </FlexItem>
            <FlexItem>
                <ClusterInitBundleDetails meta={meta} />
            </FlexItem>
        </Flex>
    );
}

export type ClusterInitBundleFormResponseMessage = {
    message: string;
    isError: boolean;
    responseData?: {
        response: ClusterInitBundleResponse;
    };
};

export type ClusterInitBundleFormMessageAlertProps = {
    message: ClusterInitBundleFormResponseMessage;
};

function ClusterInitBundleFormMessageAlert({
    message,
}: ClusterInitBundleFormMessageAlertProps): ReactElement {
    return (
        <Alert isInline variant={message.isError ? 'danger' : 'success'} title={message.message}>
            {message.responseData && (
                <ClusterInitBundleResponseDetails
                    meta={message.responseData.response.meta}
                    helmValuesBundle={message.responseData.response.helmValuesBundle}
                    kubectlBundle={message.responseData.response.kubectlBundle}
                />
            )}
        </Alert>
    );
}

export default ClusterInitBundleFormMessageAlert;
