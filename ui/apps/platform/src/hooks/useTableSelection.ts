import React from 'react';

export type UseTableSelection = {
    selected: boolean[];
    allRowsSelected: boolean;
    numSelected: number;
    hasSelections: boolean;
    onSelect: (
        event: React.FormEvent<HTMLInputElement>,
        isSelected: boolean,
        rowId: number
    ) => void;
    onSelectAll: (event: React.FormEvent<HTMLInputElement>, isSelected: boolean) => void;
    onClearAll: () => void;
    onResetAll: () => void;
    getSelectedIds: () => string[];
};

type Base = {
    id: string;
};

const defaultPreSelectedFunc = () => false;

function useTableSelection<T extends Base>(
    data: T[],
    // determines whether value should be pre-selected or not
    preSelectedFunc: (T) => boolean = defaultPreSelectedFunc
): UseTableSelection {
    const [selected, setSelected] = React.useState(data.map(preSelectedFunc));
    const allRowsSelected = selected.length !== 0 && selected.every((val) => val);
    const numSelected = selected.reduce((acc, sel) => (sel ? acc + 1 : acc), 0);
    const hasSelections = numSelected > 0;

    React.useEffect(() => {
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
            if (selected[i] && data[i]?.id) {
                ids.push(data[i].id);
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
