import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import { Button, Stack, Tooltip } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';

export type SummaryCountProps = {
    count: number;
    href: string;
    noun: string;
    tooltip?: string;
};

function SummaryCount({ count, href, noun, tooltip }: SummaryCountProps): ReactElement {
    const content = (
        <Button variant="link" component={LinkShim} href={href}>
            <Stack className="pf-v5-u-px-xs pf-v5-u-px-sm-on-xl pf-v5-u-align-items-center">
                <span className="pf-v5-u-font-size-lg-on-md pf-v5-u-font-size-sm pf-v5-u-font-weight-bold">
                    {count}
                </span>
                <span className="pf-v5-u-font-size-md-on-md pf-v5-u-font-size-xs">
                    {pluralize(noun, count)}
                </span>
            </Stack>
        </Button>
    );
    if (tooltip) {
        return (
            <Tooltip content={<div>{tooltip}</div>} position="bottom">
                {content}
            </Tooltip>
        );
    }
    return content;
}

export default SummaryCount;
