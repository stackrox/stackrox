import React from 'react';
import { useFormikContext } from 'formik';
import {
    Card,
    CardHeader,
    CardTitle,
    CardActions,
    CardBody,
    Button,
    Divider,
} from '@patternfly/react-core';
import { PencilAltIcon, TrashIcon } from '@patternfly/react-icons';

import { Policy } from 'types/policy.proto';

function PolicySection({ sectionIndex }) {
    const { values, setFieldValue } = useFormikContext<Policy>();
    const { sectionName, policyGroups } = values.policySections[sectionIndex];
    return (
        <div>
            PolicySection {sectionIndex}
            <Card isFlat>
                <CardHeader>
                    <CardTitle>{sectionName}</CardTitle>
                    <CardActions>
                        <Button variant="plain">
                            <PencilAltIcon />
                        </Button>
                        <Divider component="div" isVertical />
                        <Button variant="plain">
                            <TrashIcon />
                        </Button>
                    </CardActions>
                </CardHeader>
                <CardBody>hello</CardBody>
            </Card>
        </div>
    );
}

export default PolicySection;
