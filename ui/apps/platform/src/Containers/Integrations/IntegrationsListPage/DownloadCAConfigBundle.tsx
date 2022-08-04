import React, { ReactElement, useState } from 'react';
import { Button, Tooltip } from '@patternfly/react-core';
import FileSaver from 'file-saver';

import { fetchCAConfig } from 'services/ClustersService';
import useNotifications from 'hooks/useNotifications';

function DownloadCAConfigBundle(): ReactElement {
    const [downloadingCAConfig, setDownloadingCAConfig] = useState<boolean>(false);
    const addNotification = useNotifications();

    function onFetchCAConfig() {
        setDownloadingCAConfig(true);
        fetchCAConfig()
            .then((response) => {
                if (!response.helmValuesBundle) {
                    throw Error('server returned no data');
                }
                const bytes = atob(response.helmValuesBundle);
                const file = new Blob([bytes], {
                    type: 'application/x-yaml',
                });
                FileSaver.saveAs(file, 'ca-config.yaml');
            })
            .catch((err: { message: string }) => {
                addNotification(
                    `Problem downloading the CA config. Please try again. (${err.message})`
                );
            })
            .finally(() => {
                setDownloadingCAConfig(false);
            });
    }

    return (
        <Tooltip content={<div>Use with pre-created secrets</div>}>
            <Button
                variant="secondary"
                onClick={onFetchCAConfig}
                disabled={downloadingCAConfig}
                isLoading={downloadingCAConfig}
            >
                Download CA config
            </Button>
        </Tooltip>
    );
}

export default DownloadCAConfigBundle;
