import { Alert } from '@patternfly/react-core';
import type { FormikErrors } from 'formik';
import type { PolicySection } from 'types/policy.proto';

type PolicySectionValidationErrorProps = {
    sectionIndex: number;
    errors: string | string[] | FormikErrors<PolicySection>[];
    className?: string;
};

export function PolicySectionValidationError({
    sectionIndex,
    errors,
    className,
}: PolicySectionValidationErrorProps) {
    if (Array.isArray(errors) && typeof errors[sectionIndex] === 'string') {
        return (
            <Alert
                title={errors[sectionIndex]}
                variant="warning"
                component="p"
                isInline
                className={className}
            />
        );
    }

    return null;
}
