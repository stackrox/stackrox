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
    onCancelAll: () => void;
    getSelectedIds: () => string[];
};

type Base = {
    id: string;
};

function useTableSelection<T extends Base>(
    data: T[],
    // determines whether value should be pre-selected or not
    preSelectedFunc: (T) => boolean = () => false
): UseTableSelection {
    const [allRowsSelected, setAllRowsSelected] = React.useState(
        data.map(preSelectedFunc).every((val) => val)
    );
    const [selected, setSelected] = React.useState(data.map(preSelectedFunc));
    const numSelected = selected.reduce((acc, sel) => (sel ? acc + 1 : acc), 0);
    const hasSelections = numSelected > 0;

    React.useEffect(() => {
        setSelected(data.map(preSelectedFunc));
        setAllRowsSelected(data.map(preSelectedFunc).every((val) => val));
    }, [data]);

    const onClearAll = () => {
        setSelected(data.map(() => false));
        setAllRowsSelected(false);
    };

    const onCancelAll = () => {
        setSelected(data.map(preSelectedFunc));
        setAllRowsSelected(data.map(preSelectedFunc).every((val) => val));
    };

    const onSelect = (event, isSelected: boolean, rowId: number) => {
        setSelected(
            selected.map((sel: boolean, index: number) => (index === rowId ? isSelected : sel))
        );
        if (!isSelected && allRowsSelected) {
            setAllRowsSelected(false);
        } else if (isSelected && !allRowsSelected) {
            let allSelected = true;
            for (let i = 0; i < selected.length; i += 1) {
                if (i !== rowId) {
                    if (!selected[i]) {
                        allSelected = false;
                    }
                }
            }
            if (allSelected) {
                setAllRowsSelected(true);
            }
        }
    };

    function onSelectAll(event, isSelected: boolean) {
        setAllRowsSelected(isSelected);
        setSelected(selected.map(() => isSelected));
    }

    function getSelectedIds() {
        const ids: string[] = [];
        for (let i = 0; i < selected.length; i += 1) {
            if (selected[i]) {
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
        onCancelAll,
        getSelectedIds,
    };
}

export default useTableSelection;
