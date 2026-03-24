import { Th } from '@patternfly/react-table';
import type { ThProps } from '@patternfly/react-table';

const expandButtonWidth = '1em';
const expandButtonPaddingX = 'var(--pf-t--global--spacer--md)';
const firstTableCellPadding = 'var(--pf-v6-c-table--cell--PaddingBlockStart)';

export default function ExpandRowTh(props: ThProps) {
    return (
        <Th
            {...props}
            screenReaderText="Row expansion"
            style={{
                // Setting a defined width here prevents column shift when the table is in a loading state
                width: `calc(${expandButtonWidth} + ${expandButtonPaddingX} + ${firstTableCellPadding})`,
                ...props.style,
            }}
        />
    );
}
