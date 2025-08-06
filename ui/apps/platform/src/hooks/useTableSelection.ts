import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';

export type UseTableSelection = {
    selected: boolean[];
    allRowsSelected: boolean;
    numSelected: number;
    hasSelections: boolean;
    onSelect: (event: FormEvent<HTMLInputElement>, isSelected: boolean, rowId: number) => void;
    onSelectAll: (event: FormEvent<HTMLInputElement>, isSelected: boolean) => void;
    onClearAll: () => void;
    onResetAll: () => void;
    getSelectedIds: () => string[];
};

type Base = {
    [key: string]: unknown;
};

const defaultPreSelectedFunc = () => false;

function useTableSelection<T extends Base>(
    data: T[],
    preSelectedFunc: (item: T) => boolean = defaultPreSelectedFunc,
    identifierKey: keyof T = 'id'
): UseTableSelection {
    const [selected, setSelected] = useState(data.map(preSelectedFunc));
    const allRowsSelected = selected.length !== 0 && selected.every((val) => val);
    const numSelected = selected.reduce((acc, sel) => (sel ? acc + 1 : acc), 0);
    const hasSelections = numSelected > 0;

    useEffect(() => {
        setSelected(data.map(preSelectedFunc));
    }, [data, preSelectedFunc]);

    const onClearAll = () => {
        setSelected(data.map(() => false));
    };

    const onResetAll = () => {
        setSelected(data.map(preSelectedFunc));
    };

    const onSelect = (event, isSelected: boolean, rowId: number) => {
        setSelected(
            selected.map((sel: boolean, index: number) => (index === rowId ? isSelected : sel))
        );
    };

    function onSelectAll(event, isSelected: boolean) {
        setSelected(selected.map(() => isSelected));
    }

    function getSelectedIds() {
        const ids: string[] = [];
        for (let i = 0; i < selected.length; i += 1) {
            if (selected[i] && data[i]?.[identifierKey]) {
                ids.push(String(data[i][identifierKey]));
            }
        }
        return ids;
    }

    return {
        selected,
        allRowsSelected,
        numSelected,
        hasSelections,
        onSelect,
        onSelectAll,
        onClearAll,
        onResetAll,
        getSelectedIds,
    };
}

export default useTableSelection;
