import { Alert } from '@patternfly/react-core';

type OutdatedDataBannerProps = {
    outdatedClusterCount: number;
};

function OutdatedDataBanner({ outdatedClusterCount }: OutdatedDataBannerProps) {
    if (outdatedClusterCount <= 0) {
        return null;
    }
    return (
        <Alert
            variant="warning"
            isInline
            title={`${outdatedClusterCount} cluster${outdatedClusterCount > 1 ? 's have' : ' has'} outdated compliance data`}
            component="p"
        >
            Some compliance results may not reflect the latest scan cycle. Check the cluster
            status for details.
        </Alert>
    );
}

export default OutdatedDataBanner;
