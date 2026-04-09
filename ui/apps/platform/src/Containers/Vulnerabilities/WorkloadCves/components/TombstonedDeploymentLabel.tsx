import { Label, Popover } from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';

import PopoverBodyContent from 'Components/PopoverBodyContent';
import { getDateTime } from 'utils/dateUtils';

export type TombstonedDeploymentLabelProps = {
    deletedAt: string;
    isCompact?: boolean;
    variant?: 'outline' | 'filled';
};

function TombstonedDeploymentLabel({
    deletedAt,
    isCompact,
    variant,
}: TombstonedDeploymentLabelProps) {
    return (
        <Popover
            aria-label="Deleted deployment"
            bodyContent={
                <PopoverBodyContent
                    headerContent="Deployment deleted"
                    bodyContent={`This deployment was deleted on ${getDateTime(deletedAt)}.`}
                />
            }
            enableFlip
            hasAutoWidth
            position="top"
        >
            <Label
                color="grey"
                isCompact={isCompact}
                variant={variant}
                icon={<TrashIcon />}
                style={{ cursor: 'pointer' }}
            >
                Deleted
            </Label>
        </Popover>
    );
}

export default TombstonedDeploymentLabel;
