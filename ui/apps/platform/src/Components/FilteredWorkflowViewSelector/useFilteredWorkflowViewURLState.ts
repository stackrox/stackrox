import useURLStringUnion from 'hooks/useURLStringUnion';
import { FilteredWorkflowView, filteredWorkflowViews } from './types';

export type FilteredWorkflowViewURLStateResult = {
    filteredWorkflowView: FilteredWorkflowView;
    setFilteredWorkflowView: (value: FilteredWorkflowView) => void;
};

function useFilteredWorkflowViewURLState(): FilteredWorkflowViewURLStateResult {
    const [filteredWorkflowView, setFilteredWorkflowView] = useURLStringUnion(
        'filteredWorkflowView',
        filteredWorkflowViews
    );

    return {
        filteredWorkflowView,
        setFilteredWorkflowView,
    };
}

export default useFilteredWorkflowViewURLState;
