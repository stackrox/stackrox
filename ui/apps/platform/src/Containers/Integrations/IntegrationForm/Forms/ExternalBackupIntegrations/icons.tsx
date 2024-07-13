/* eslint-disable no-void */
import React, { ReactElement } from 'react';
import { Button } from '@patternfly/react-core';

import IntegrationHelpIcon from '../Components/IntegrationHelpIcon';

export function gcsWorkloadIdentity(): ReactElement {
    return (
        <IntegrationHelpIcon
            helpTitle="GCP workload identity"
            helpText={
                <div>
                    Enables authentication via short-lived tokens using GCP workload identities. See
                    the{' '}
                    <Button variant="link" isInline>
                        <a
                            href="https://docs.openshift.com/acs/integration/integrate-using-short-lived-tokens.html"
                            target="_blank"
                            rel="noreferrer"
                        >
                            Red Hat ACS documentation
                        </a>
                    </Button>{' '}
                    for more information.
                </div>
            }
            ariaLabel="Help for short-lived tokens"
        />
    );
}

export function s3EndpointIcon(): ReactElement {
    return (
        <IntegrationHelpIcon
            helpTitle="AWS S3 endpoint"
            helpText={
                <div>
                    Modifies the endpoint under which S3 is reached. Note that when using a non-AWS
                    service provider, it is recommended to create an <em>S3 API Compatible</em>{' '}
                    integration instead. See the{' '}
                    <Button variant="link" isInline>
                        <a
                            href="https://docs.aws.amazon.com/general/latest/gr/s3.html"
                            target="_blank"
                            rel="noreferrer"
                        >
                            AWS S3 documentation
                        </a>
                    </Button>{' '}
                    for more information.
                </div>
            }
            ariaLabel="Help for AWS S3 endpoint"
        />
    );
}

export function s3RegionIcon(): ReactElement {
    return (
        <IntegrationHelpIcon
            helpTitle="AWS S3 region"
            helpText={
                <div>
                    Modifies the endpoint under which S3 is reached. Note that when using a non-AWS
                    service provider, it is recommended to create an <em>S3 API Compatible</em>{' '}
                    integration instead. See the{' '}
                    <Button variant="link" isInline>
                        <a
                            href="https://docs.aws.amazon.com/general/latest/gr/s3.html"
                            target="_blank"
                            rel="noreferrer"
                        >
                            AWS S3 documentation
                        </a>
                    </Button>{' '}
                    for a complete list of AWS regions.
                </div>
            }
            ariaLabel="Help for AWS S3 region"
        />
    );
}

export function s3IamRole(): ReactElement {
    return (
        <IntegrationHelpIcon
            helpTitle="AWS container IAM role"
            helpText={
                <div>
                    Enables authentication via short-lived tokens using AWS Secure Token Service.
                    See the{' '}
                    <Button variant="link" isInline>
                        <a
                            href="https://docs.openshift.com/acs/integration/integrate-using-short-lived-tokens.html"
                            target="_blank"
                            rel="noreferrer"
                        >
                            Red Hat ACS documentation
                        </a>
                    </Button>{' '}
                    for more information.
                </div>
            }
            ariaLabel="Help for short-lived tokens"
        />
    );
}

export function s3CompatibleEndpointIcon(): ReactElement {
    return (
        <IntegrationHelpIcon
            helpTitle="Endpoint"
            helpText={
                <div>
                    Modifies the endpoint under which the S3 compatible service is reached. Must be
                    reachable via https. Note that when using AWS S3, it is recommended to create an{' '}
                    <em>Amazon S3</em> integration instead.
                </div>
            }
            ariaLabel="Help for endpoint"
        />
    );
}

export function s3CompatibleRegionIcon(): ReactElement {
    return (
        <IntegrationHelpIcon
            helpTitle="Region"
            helpText={
                <div>
                    Consult the service provider&apos;s S3 compatibility instructions for the
                    correct region.
                </div>
            }
            ariaLabel="Help for region"
        />
    );
}

export function objectPrefixIcon(): ReactElement {
    return (
        <IntegrationHelpIcon
            helpTitle="Object prefix"
            helpText={
                <div>
                    Creates a new folder &#60;prefix&#62; under which backups files are placed.
                </div>
            }
            ariaLabel="Help for object prefix"
        />
    );
}

export function urlStyleIcon(): ReactElement {
    return (
        <IntegrationHelpIcon
            helpTitle="Virtual hosting of buckets"
            helpText={
                <div>
                    Defines the bucket URL addressing. Virtual-hosted-style buckets are addressed as
                    https://&#60;bucket&#62;.&#60;endpoint&#62; while path-style buckets are
                    addressed as https://&#60;endpoint&#62;/&#60;bucket&#62;. See the{' '}
                    <Button variant="link" isInline>
                        <a
                            href="https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html"
                            target="_blank"
                            rel="noreferrer"
                        >
                            AWS documentation about virtual hosting
                        </a>
                    </Button>{' '}
                    for more information.
                </div>
            }
            ariaLabel="Help for URL style"
        />
    );
}
