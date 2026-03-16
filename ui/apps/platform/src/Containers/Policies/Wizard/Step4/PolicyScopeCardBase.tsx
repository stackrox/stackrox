import type { ReactElement, ReactNode } from 'react';
import { Button, Card, CardBody, CardHeader, CardTitle } from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';

type PolicyScopeCardBaseProps = {
    title: string;
    onDelete: () => void;
    children: ReactNode;
};

function PolicyScopeCardBase({
    title,
    onDelete,
    children,
}: PolicyScopeCardBaseProps): ReactElement {
    return (
        <Card isCompact>
            <CardHeader
                actions={{
                    actions: (
                        <Button
                            variant="plain"
                            onClick={onDelete}
                            title={`Delete ${title}`}
                            icon={<TrashIcon />}
                        />
                    ),
                }}
            >
                <CardTitle>{title}</CardTitle>
            </CardHeader>
            <CardBody>{children}</CardBody>
        </Card>
    );
}

export default PolicyScopeCardBase;
