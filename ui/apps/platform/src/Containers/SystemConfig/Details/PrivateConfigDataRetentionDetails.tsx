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

import { PrivateConfig } from 'types/config.proto';

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

export type PrivateConfigDataRetentionDetailsProps = {
    privateConfig: PrivateConfig;
};

const PrivateConfigDataRetentionDetails = ({
    privateConfig,
}: PrivateConfigDataRetentionDetailsProps): ReactElement => {
    return (
        <Card data-testid="data-retention-config">
            <CardTitle>Data retention configuration</CardTitle>
            <Divider component="div" />
            <CardBody>
                <Gallery hasGutter>
                    <GalleryItem>
                        <NumberBox
                            label="All runtime violations"
                            value={privateConfig?.alertConfig?.allRuntimeRetentionDurationDays}
                            suffix="day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Runtime violations for deleted deployments"
                            value={privateConfig?.alertConfig?.deletedRuntimeRetentionDurationDays}
                            suffix="day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Resolved deploy-phase violations"
                            value={privateConfig?.alertConfig?.resolvedDeployRetentionDurationDays}
                            suffix="day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Attempted deploy-phase violations"
                            value={privateConfig?.alertConfig?.attemptedDeployRetentionDurationDays}
                            suffix="day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Attempted runtime violations"
                            value={
                                privateConfig?.alertConfig?.attemptedRuntimeRetentionDurationDays
                            }
                            suffix="day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Images no longer deployed"
                            value={privateConfig?.imageRetentionDurationDays}
                            suffix="day"
                        />
                    </GalleryItem>
                    <GalleryItem>
                        <NumberBox
                            label="Expired vulnerability requests"
                            value={privateConfig?.expiredVulnReqRetentionDurationDays}
                            suffix="day"
                        />
                    </GalleryItem>
                </Gallery>
            </CardBody>
        </Card>
    );
};

export default PrivateConfigDataRetentionDetails;
