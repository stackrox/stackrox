import { Tooltip } from '@patternfly/react-core';
import type { TooltipProps } from '@patternfly/react-core';

export type ConditionalTooltipProps = TooltipProps & {
    renderTooltip: boolean;
};

function ConditionalTooltip({ renderTooltip, children, ...props }: ConditionalTooltipProps) {
    if (renderTooltip) {
        return <Tooltip {...props}>{children}</Tooltip>;
    }
    return children;
}

export default ConditionalTooltip;
