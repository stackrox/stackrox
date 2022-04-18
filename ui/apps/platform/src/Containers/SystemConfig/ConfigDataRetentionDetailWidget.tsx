import React, { ReactElement } from 'react';
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

import { PrivateConfig } from './SystemConfigTypes';

const UNKNOWN_FLAG = -1;

type NumberBoxProps = {
    label: string;
    value?: number;
    suffix?: string;
};

const NumberBox = ({ label, value = UNKNOWN_FLAG, suffix = '' }: NumberBoxProps): ReactElement => (
    <Hint data-testid="number-box" className="pf-u-h-100">
        <HintTitle className="pf-u-font-size-sm">{label}</HintTitle>
        <HintBody className="pf-u-font-size-xl pf-u-font-weight-bold">
            {value === UNKNOWN_FLAG && `Unknown`}
            {!value && `Never deleted`}
            {value > 0 && `${value} ${pluralize(suffix, value)}`}
        </HintBody>
    </Hint>
);

export type DataRetentionDetailWidgetProps = {
    privateConfig: PrivateConfig;
};

const DataRetentionDetailWidget = ({
    privateConfig,
}: DataRetentionDetailWidgetProps): ReactElement => {
    return (
        <Card data-testid="data-retention-config">
            <CardTitle>Data Retention Configuration</CardTitle>
            <Divider component="div" />
            <CardBody>
                <Gallery hasGutter>
                    <GalleryItem>
                        <NumberBox
                            label="All Runtime Violations"
                            value={privateConfig?.alertConfig?.allRuntimeRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Runtime Violations For Deleted Deployments"
                            value={privateConfig?.alertConfig?.deletedRuntimeRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Resolved Deploy-Phase Violations"
                            value={privateConfig?.alertConfig?.resolvedDeployRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Attempted Deploy-Phase Violations"
                            value={privateConfig?.alertConfig?.attemptedDeployRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Attempted Runtime Violations"
                            value={
                                privateConfig?.alertConfig?.attemptedRuntimeRetentionDurationDays
                            }
                            suffix="Day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Images No Longer Deployed"
                            value={privateConfig?.imageRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Expired Vulnerability Requests"
                            value={privateConfig?.expiredVulnReqRetentionDurationDays}
                            suffix="Day"
                        />
                    </GalleryItem>
                </Gallery>
            </CardBody>
        </Card>
    );
};

export default DataRetentionDetailWidget;
