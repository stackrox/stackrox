import React, { ReactElement, forwardRef, useRef, useEffect } from 'react';

export type IndeterminateCheckboxProps = {
    title: string;
    checked: boolean;
    indeterminate: boolean;
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
};

const useCombinedRefs = (...refs): React.MutableRefObject<HTMLInputElement | null> => {
    const targetRef = React.useRef(null);

    React.useEffect(() => {
        refs.forEach((ref) => {
            if (!ref) {
                return;
            }

            if (typeof ref === 'function') {
                ref(targetRef.current);
            } else {
                const modifiedRef = ref;
                modifiedRef.current = targetRef.current;
            }
        });
    }, [refs]);

    return targetRef;
};

function IndeterminateCheckbox(
    props: IndeterminateCheckboxProps,
    ref: React.Ref<HTMLInputElement>
): ReactElement {
    const { title, checked, indeterminate, onChange } = props;
    const defaultRef = useRef<HTMLInputElement>(null);
    const combinedRef = useCombinedRefs(ref, defaultRef);

    useEffect(() => {
        if (combinedRef.current !== null) {
            combinedRef.current.indeterminate = indeterminate ?? false;
        }
    }, [combinedRef, indeterminate]);

    return (
        <input
            aria-label="checkbox"
            type="checkbox"
            className="form-checkbox h-4 w-4 border-base-500 text-primary-500"
            ref={combinedRef}
            title={title}
            checked={checked}
            onChange={onChange}
        />
    );
}

const IndeterminateCheckboxWithForwardRef = forwardRef<
    HTMLInputElement,
    IndeterminateCheckboxProps
>(IndeterminateCheckbox);

export default IndeterminateCheckboxWithForwardRef;
