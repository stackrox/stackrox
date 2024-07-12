/* eslint-disable no-void */
import React, { ReactElement } from 'react';
import { Popover } from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';

export function gcsWorkloadIdentity(): ReactElement {
    return (
        <Popover
            bodyContent={
                <div>
                    <a
                        href="https://docs.openshift.com/acs/integration/integrate-using-short-lived-tokens.html"
                        target="_blank"
                        rel="noreferrer"
                    >
                        Enables authentication via short-lived tokens using GCP workload identities.
                        See the Red Hat RHACS documentation for more information.
                    </a>
                </div>
            }
            headerContent={'GCP workload identity'}
        >
            <button
                type="button"
                aria-label="More info for input"
                onClick={(e) => e.preventDefault()}
                aria-describedby="simple-form-name-01"
                className="pf-v5-c-form__group-label-help"
            >
                <HelpIcon />
            </button>
        </Popover>
    );
}

export function s3EndpointIcon(): ReactElement {
    return (
        <Popover
            bodyContent={
                <div>
                    <a
                        href="https://docs.aws.amazon.com/general/latest/gr/s3.html"
                        target="_blank"
                        rel="noreferrer"
                    >
                        Modifies the endpoint under which S3 is reached. Note that when using a
                        non-AWS service provider, it is recommended to create an *S3 API Compatible*
                        integration instead. See the AWS documentation about S3 endpoints for a
                        complete list of AWS endpoints.
                    </a>
                </div>
            }
            headerContent={'AWS S3 endpoint'}
        >
            <button
                type="button"
                aria-label="More info for input"
                onClick={(e) => e.preventDefault()}
                aria-describedby="simple-form-name-01"
                className="pf-v5-c-form__group-label-help"
            >
                <HelpIcon />
            </button>
        </Popover>
    );
}

export function s3RegionIcon(): ReactElement {
    return (
        <Popover
            bodyContent={
                <div>
                    <a
                        href="https://docs.aws.amazon.com/general/latest/gr/s3.html"
                        target="_blank"
                        rel="noreferrer"
                    >
                        See the AWS documentation about S3 endpoints for a complete list of AWS
                        regions.
                    </a>
                </div>
            }
            headerContent={'AWS S3 region'}
        >
            <button
                type="button"
                aria-label="More info for input"
                onClick={(e) => e.preventDefault()}
                aria-describedby="simple-form-name-01"
                className="pf-v5-c-form__group-label-help"
            >
                <HelpIcon />
            </button>
        </Popover>
    );
}

export function s3IamRole(): ReactElement {
    return (
        <Popover
            bodyContent={
                <div>
                    <a
                        href="https://docs.openshift.com/acs/integration/integrate-using-short-lived-tokens.html"
                        target="_blank"
                        rel="noreferrer"
                    >
                        Enables authentication via short-lived tokens using AWS Secure Token
                        Service. See the Red Hat RHACS documentation for more information.
                    </a>
                </div>
            }
            headerContent={'AWS container IAM role'}
        >
            <button
                type="button"
                aria-label="More info for input"
                onClick={(e) => e.preventDefault()}
                aria-describedby="simple-form-name-01"
                className="pf-v5-c-form__group-label-help"
            >
                <HelpIcon />
            </button>
        </Popover>
    );
}

export function s3CompatibleEndpointIcon(): ReactElement {
    return (
        <Popover
            bodyContent={
                <div>
                    Modifies the endpoint under which the S3 compatible service is reached. Must be
                    reachable via https.
                </div>
            }
            headerContent={'Endpoint'}
        >
            <button
                type="button"
                aria-label="More info for input"
                onClick={(e) => e.preventDefault()}
                aria-describedby="simple-form-name-01"
                className="pf-v5-c-form__group-label-help"
            >
                <HelpIcon />
            </button>
        </Popover>
    );
}

export function s3CompatibleRegionIcon(): ReactElement {
    return (
        <Popover
            bodyContent={
                <div>
                    Consult the service provider&apos;s S3 compatibility instructions for the
                    correct region.
                </div>
            }
            headerContent={'Region'}
        >
            <button
                type="button"
                aria-label="More info for input"
                onClick={(e) => e.preventDefault()}
                aria-describedby="simple-form-name-01"
                className="pf-v5-c-form__group-label-help"
            >
                <HelpIcon />
            </button>
        </Popover>
    );
}

export function objectPrefixIcon(): ReactElement {
    return (
        <Popover
            bodyContent={
                <div>
                    Creates a new folder &#60;prefix&#62; under which backups files are placed.
                </div>
            }
            headerContent={'Object prefix'}
        >
            <button
                type="button"
                aria-label="More info for input"
                onClick={(e) => e.preventDefault()}
                aria-describedby="simple-form-name-01"
                className="pf-v5-c-form__group-label-help"
            >
                <HelpIcon />
            </button>
        </Popover>
    );
}

export function urlStyleIcon(): ReactElement {
    return (
        <Popover
            bodyContent={
                <div>
                    <a
                        href="https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html"
                        target="_blank"
                        rel="noreferrer"
                    >
                        Defines the bucket URL addressing. Virtual-hosted-style buckets are
                        addressed as https://&#60;bucket&#62;.&#60;endpoint&#62; while path-style
                        buckets are addressed as https://&#60;endpoint&#62;/&#60;bucket&#62;. See
                        the AWS documentation about virtual hosting for more information.
                    </a>
                </div>
            }
            headerContent={'Virtual hosting of buckets'}
        >
            <button
                type="button"
                aria-label="More info for input"
                onClick={(e) => e.preventDefault()}
                aria-describedby="simple-form-name-01"
                className="pf-v5-c-form__group-label-help"
            >
                <HelpIcon />
            </button>
        </Popover>
    );
}
