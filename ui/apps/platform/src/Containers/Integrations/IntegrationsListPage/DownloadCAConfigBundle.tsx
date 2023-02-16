import React, { ReactElement, useState } from 'react';
import { useDispatch } from 'react-redux';
import { Button, Tooltip } from '@patternfly/react-core';
import FileSaver from 'file-saver';

import { actions } from 'reducers/notifications';
import { fetchCAConfig } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

function DownloadCAConfigBundle(): ReactElement {
    const [downloadingCAConfig, setDownloadingCAConfig] = useState<boolean>(false);
    const dispatch = useDispatch();

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
            .catch((error) => {
                const errorMessage = getAxiosErrorMessage(error);
                dispatch(
                    actions.addNotification(
                        `Problem downloading the CA config. Please try again. (${errorMessage})`
                    )
                );
                setTimeout(() => {
                    dispatch(actions.removeOldestNotification());
                }, 5000);
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
