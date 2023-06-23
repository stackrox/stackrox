import useURLParameter, { QueryValue } from 'hooks/useURLParameter';

type PageActionResult<T> = {
    pageAction: T | undefined;
    setPageAction: (action: T) => void;
};

function usePageAction<T>(): PageActionResult<T> {
    const [pageActionParam, setPageActionParam] = useURLParameter('action', undefined);

    const pageAction = pageActionParam as T;

    function setPageAction(action: T): void {
        setPageActionParam(action as QueryValue);
    }

    return { pageAction, setPageAction };
}

export default usePageAction;
