import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import {
    Card,
    CardBody,
    CardTitle,
    Divider,
    Gallery,
    GalleryItem,
    Hint,
    HintTitle,
    HintBody,
} from '@patternfly/react-core';

const UNKNOWN_FLAG = -1;

const NumberBox = ({ label, value, suffix }) => (
    <Hint data-testid="number-box" className="pf-u-h-100">
        <HintTitle className="pf-u-font-size-sm">{label}</HintTitle>
        <HintBody className="pf-u-font-size-xl pf-u-font-weight-bold">
            {value === UNKNOWN_FLAG && `Unknown`}
            {!value && `Never deleted`}
            {value > 0 && `${value} ${pluralize(suffix, value)}`}
        </HintBody>
    </Hint>
);

NumberBox.propTypes = {
    label: PropTypes.string.isRequired,
    value: PropTypes.number,
    suffix: PropTypes.string,
};

NumberBox.defaultProps = {
    suffix: '',
    value: UNKNOWN_FLAG,
};

const DataRetentionDetailWidget = ({ config }) => {
    // safeguard, because on initial navigate, some nested objects are not loaded yet
    const privateConfig = config.privateConfig || {};
    const alertConfig = privateConfig.alertConfig || {};

    return (
        <Card data-testid="data-retention-config">
            <CardTitle>Data Retention Configuration</CardTitle>
            <Divider component="div" />
            <CardBody>
                <Gallery hasGutter>
                    <GalleryItem>
                        <NumberBox
                            label="All Runtime Violations"
                            value={alertConfig.allRuntimeRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Runtime Violations For Deleted Deployments"
                            value={alertConfig.deletedRuntimeRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Resolved Deploy-Phase Violations"
                            value={alertConfig.resolvedDeployRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Attempted Deploy-Phase Violations"
                            value={alertConfig.attemptedDeployRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Attempted Runtime Violations"
                            value={alertConfig.attemptedRuntimeRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Images No Longer Deployed"
                            value={privateConfig.imageRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                </Gallery>
            </CardBody>
        </Card>
    );
};

DataRetentionDetailWidget.propTypes = {
    config: PropTypes.shape({
        publicConfig: PropTypes.shape({
            loginNotice: PropTypes.shape({}),
        }),
        privateConfig: PropTypes.shape({}),
    }).isRequired,
};

export default DataRetentionDetailWidget;
