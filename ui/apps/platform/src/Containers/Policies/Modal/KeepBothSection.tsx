import { Field } from 'formik';
import { Radio } from '@patternfly/react-core';

export type KeepBothSectionProps = {
    changeRadio: (handler, name: string, value: string) => () => void;
};

const KeepBothSection = ({ changeRadio }: KeepBothSectionProps) => {
    return (
        <Field name="resolution">
            {({ field }) => (
                <Radio
                    name={field.name}
                    id="keep-both-radio"
                    value="keepBoth"
                    checked={field.value === 'keepBoth'}
                    onChange={changeRadio(field.onChange, field.name, 'keepBoth')}
                    label="Keep both policies (imported policy will be assigned a new ID)"
                />
            )}
        </Field>
    );
};

export default KeepBothSection;
