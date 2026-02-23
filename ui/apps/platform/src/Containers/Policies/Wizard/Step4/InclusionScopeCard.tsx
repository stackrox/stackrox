import type { FormEvent, ReactElement } from 'react';
import { Flex, Form, FormGroup, TextInput } from '@patternfly/react-core';

import type { PolicyScope } from 'types/policy.proto';

import PolicyScopeCardBase from './PolicyScopeCardBase';

type InclusionScopeCardProps = {
    scope: PolicyScope;
    onChange: (newScope: PolicyScope) => void;
    onDelete: () => void;
};

function InclusionScopeCard({ scope, onChange, onDelete }: InclusionScopeCardProps): ReactElement {
    function handleChangeLabelKey(_event: FormEvent, key: string) {
        onChange({ ...scope, label: { key, value: scope.label?.value ?? '' } });
    }

    function handleChangeLabelValue(_event: FormEvent, val: string) {
        onChange({ ...scope, label: { key: scope.label?.key ?? '', value: val } });
    }

    return (
        <PolicyScopeCardBase title="Inclusion scope" onDelete={onDelete}>
            <Form>
                <FormGroup label="Deployment label">
                    <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                        <TextInput
                            aria-label="Deployment label key"
                            onChange={handleChangeLabelKey}
                            placeholder="Label key"
                            type="text"
                            value={scope.label?.key ?? ''}
                        />
                        <TextInput
                            aria-label="Deployment label value"
                            onChange={handleChangeLabelValue}
                            placeholder="Label value"
                            type="text"
                            value={scope.label?.value ?? ''}
                        />
                    </Flex>
                </FormGroup>
            </Form>
        </PolicyScopeCardBase>
    );
}

export default InclusionScopeCard;
