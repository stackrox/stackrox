import { Label } from '@patternfly/react-core';
import type { LabelProps } from '@patternfly/react-core';

export type TechPreviewLabelProps = LabelProps;

// Render TechPreviewLabel when width is limited: in left navigation or integration tile.
// Render TechnologyPreviewLabel when when width is not limited: in heading.
function TechPreviewLabel({ className, ...props }: TechPreviewLabelProps) {
    return (
        <Label isCompact color="purple" className={className} {...props}>
            Tech preview
        </Label>
    );
}

export default TechPreviewLabel;
