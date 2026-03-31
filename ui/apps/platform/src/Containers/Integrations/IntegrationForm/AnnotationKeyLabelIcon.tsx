import type { ReactElement } from 'react';
import { FormGroupLabelHelp, Popover } from '@patternfly/react-core';

function AnnotationKeyLabelIcon(): ReactElement {
    return (
        <Popover
            aria-label="Annotation field help popover"
            showClose={false}
            bodyContent={
                <div>
                    Using an annotation key, you can define an audience to notify about policy
                    violations associated with any given deployment or namespace. If the deployment
                    and/or namespace has the annotation, its value overrides the default.
                </div>
            }
        >
            <FormGroupLabelHelp aria-label="Information about annotation field" />
        </Popover>
    );
}

export default AnnotationKeyLabelIcon;
