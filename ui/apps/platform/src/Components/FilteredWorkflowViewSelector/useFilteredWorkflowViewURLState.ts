import useURLStringUnion from 'hooks/useURLStringUnion';
import { FilteredWorkflowView, filteredWorkflowViews } from './types';

export type FilteredWorkflowViewURLStateResult = {
    filteredWorkflowView: FilteredWorkflowView;
};

export const filteredWorkflowViewKey = 'filteredWorkflowView';

function useFilteredWorkflowViewURLState(
    defaultView?: FilteredWorkflowView
): FilteredWorkflowViewURLStateResult {
    const [filteredWorkflowView] = useURLStringUnion(
        filteredWorkflowViewKey,
        filteredWorkflowViews,
        defaultView
    );

    return {
        filteredWorkflowView,
    };
}

export default useFilteredWorkflowViewURLState;
